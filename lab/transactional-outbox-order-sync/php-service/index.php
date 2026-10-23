<?php
declare(strict_types=1);

header('Content-Type: application/json');

$serviceName = getenv('SERVICE_NAME') ?: 'service';

function db(): PDO {
    static $pdo = null;
    if ($pdo === null) {
        $pdo = new PDO(
            sprintf(
                'mysql:host=%s;port=%s;dbname=%s;charset=utf8mb4',
                getenv('DB_HOST') ?: '127.0.0.1',
                getenv('DB_PORT') ?: '3306',
                getenv('DB_NAME') ?: 'orders'
            ),
            getenv('DB_USER') ?: 'app',
            getenv('DB_PASSWORD') ?: 'apppass',
            [
                PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
                PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            ]
        );
    }
    return $pdo;
}

function jsonBody(): array {
    $data = json_decode(file_get_contents('php://input') ?: '{}', true);
    return is_array($data) ? $data : [];
}

function send(int $code, array $body): void {
    http_response_code($code);
    echo json_encode($body);
    exit;
}

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$path   = rtrim(parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?? '/', '/');
if ($path === '') {
    $path = '/';
}

$isSynced = strtolower($_SERVER['HTTP_X_SYNCED'] ?? '') === 'true';

try {
    if ($method === 'GET' && $path === '/health') {
        send(200, ['status' => 'ok', 'service' => $serviceName]);
    }

    if ($method === 'GET' && $path === '/orders') {
        $stmt = db()->query(
            'SELECT id, order_number, customer_name, status, source, created_at
             FROM orders ORDER BY id DESC'
        );
        send(200, ['service' => $serviceName, 'orders' => $stmt->fetchAll()]);
    }

    if ($method === 'POST' && $path === '/orders') {
        $body = jsonBody();
        $orderNumber = $body['order_number'] ?? '';
        $customerName = $body['customer_name'] ?? '';
        if ($orderNumber === '' || $customerName === '') {
            send(400, ['error' => 'order_number and customer_name are required']);
        }

        // TRANSACTIONAL OUTBOX: the order and the outbox message are committed
        // atomically in one local transaction - no dual-write across services.
        $pdo = db();
        $pdo->beginTransaction();
        try {
            $stmt = $pdo->prepare(
                'INSERT INTO orders (order_number, customer_name, status, source)
                 VALUES (?, ?, ?, ?)
                 ON DUPLICATE KEY UPDATE
                   customer_name = VALUES(customer_name),
                   status        = VALUES(status),
                   source        = VALUES(source),
                   updated_at    = NOW()'
            );
            $stmt->execute([
                $orderNumber,
                $customerName,
                $isSynced ? 'synced' : 'new',
                $isSynced ? 'synced' : 'local',
            ]);

            // When the order arrived via sync we must NOT append another
            // outbox event, otherwise the peer would sync it back forever.
            if (!$isSynced) {
                $payload = json_encode([
                    'order_number' => $orderNumber,
                    'customer_name' => $customerName,
                ]);
                $stmt = $pdo->prepare(
                    'INSERT INTO outbox (aggregate_type, aggregate_id, type, payload)
                     VALUES (\'Order\', ?, \'OrderCreated\', ?)'
                );
                $stmt->execute([$orderNumber, $payload]);
            }

            $pdo->commit();
            send(201, [
                'service'     => $serviceName,
                'order_number' => $orderNumber,
                'source'      => $isSynced ? 'synced' : 'local',
            ]);
        } catch (PDOException $e) {
            $pdo->rollBack();
            send(500, ['error' => $e->getMessage()]);
        }
    }

    if ($method === 'GET' && $path === '/outbox') {
        $processed = $_GET['processed'] ?? null;
        $sql = 'SELECT id, aggregate_type, aggregate_id, type, payload, created_at,
                       processed_at, processing_attempts
                FROM outbox';
        $params = [];
        if ($processed === '0') {
            $sql .= ' WHERE processed_at IS NULL';
        } elseif ($processed === '1') {
            $sql .= ' WHERE processed_at IS NOT NULL';
        }
        $sql .= ' ORDER BY id ASC';
        $stmt = db()->prepare($sql);
        $stmt->execute($params);
        // PDO returns the JSON column as a string; decode it so consumers get
        // the payload as an object, not a double-encoded string.
        $rows = array_map(static function (array $row): array {
            $row['payload'] = json_decode($row['payload'], true);
            return $row;
        }, $stmt->fetchAll());
        send(200, ['service' => $serviceName, 'outbox' => $rows]);
    }

    if ($method === 'POST' && preg_match('#^/outbox/(\d+)/process$#', $path, $matches)) {
        $id = (int) $matches[1];
        $pdo = db();
        $pdo->beginTransaction();
        try {
            // Claim the row only if it is still pending (FOR UPDATE serialises
            // concurrent workers; processed_at IS NULL makes redelivery safe).
            $stmt = $pdo->prepare(
                'SELECT id FROM outbox WHERE id = ? AND processed_at IS NULL FOR UPDATE'
            );
            $stmt->execute([$id]);
            if (!$stmt->fetch()) {
                $pdo->rollBack();
                send(404, ['error' => 'outbox entry not found or already processed']);
            }
            $pdo->prepare('UPDATE outbox SET processed_at = NOW() WHERE id = ?')
                ->execute([$id]);
            $pdo->commit();
            send(200, ['service' => $serviceName, 'processed' => $id]);
        } catch (PDOException $e) {
            $pdo->rollBack();
            send(500, ['error' => $e->getMessage()]);
        }
    }

    send(404, ['error' => 'not found']);
} catch (Throwable $e) {
    send(500, ['error' => $e->getMessage()]);
}
