FROM php:8.2-apache

# PDO + pdo_mysql let PHP talk to MySQL; mod_rewrite routes every request to
# the single index.php front controller.
RUN docker-php-ext-install pdo pdo_mysql \
    && a2enmod rewrite

WORKDIR /var/www/html

COPY index.php .htaccess ./

# Allow .htaccess so the front-controller rewrite works.
RUN sed -i 's/AllowOverride None/AllowOverride All/' /etc/apache2/apache2.conf

EXPOSE 80
