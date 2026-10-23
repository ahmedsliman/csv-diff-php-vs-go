"""Generate old.csv and new.csv with millions of rows for diff testing.

Columns: EAN,price,quantity

Layout:
- 1,500,000 common product IDs (present in both files)
-   500,000 old-only IDs   (present in old, missing in new  -> removed)
-   500,000 new-only IDs   (present in new, missing in old  -> new)
-  300,000 of the common IDs are modified in new.csv only   -> modified

Old total: 2,000,000 rows
New total: 2,000,000 rows

Outputs:
  data/old.csv
  data/new.csv
"""

import csv
import os
import random

DATA_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "data")
OLD_PATH = os.path.join(DATA_DIR, "old.csv")
NEW_PATH = os.path.join(DATA_DIR, "new.csv")

COMMON_COUNT = 5_000_000
OLD_ONLY_COUNT = 1_000_000
NEW_ONLY_COUNT = 1_000_000
MODIFIED_COUNT = 500_000  # subset of COMMON_COUNT changed in new.csv only

RANDOM_SEED = 42


def rand_price():
    return round(random.uniform(1.00, 999.99), 2)


def rand_quantity():
    return random.randint(0, 10_000)


def rand_ean():
    # 13-digit EAN-like numeric string
    return str(random.randint(10_000_000_000_00, 99_999_999_999_99))


def main():
    random.seed(RANDOM_SEED)
    os.makedirs(DATA_DIR, exist_ok=True)

    # --- Generate ID pools -----------------------------------------------
    common_ids = [rand_ean() for _ in range(COMMON_COUNT)]
    old_only_ids = [rand_ean() for _ in range(OLD_ONLY_COUNT)]
    new_only_ids = [rand_ean() for _ in range(NEW_ONLY_COUNT)]

    # Pick which common IDs will be modified in the new file only
    modified_ids = set(random.sample(common_ids, MODIFIED_COUNT))

    # --- Build common product attributes once, reused across files -------
    # Store (price, quantity) per common EAN so non-modified rows stay
    # identical between old.csv and new.csv.
    common_attrs = {
        ean: (rand_price(), rand_quantity()) for ean in common_ids
    }

    # --- Old file: common + old-only -------------------------------------
    with open(OLD_PATH, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["EAN", "price", "quantity"])
        for ean in common_ids:
            price, qty = common_attrs[ean]
            writer.writerow([ean, price, qty])
        for ean in old_only_ids:
            writer.writerow([ean, rand_price(), rand_quantity()])

    # --- New file: common (some modified) + new-only ---------------------
    with open(NEW_PATH, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["EAN", "price", "quantity"])
        for ean in common_ids:
            if ean in modified_ids:
                writer.writerow([ean, rand_price(), rand_quantity()])
            else:
                price, qty = common_attrs[ean]
                writer.writerow([ean, price, qty])
        for ean in new_only_ids:
            writer.writerow([ean, rand_price(), rand_quantity()])

    print(f"Old file:  {OLD_PATH}  ({COMMON_COUNT + OLD_ONLY_COUNT:,} rows)")
    print(f"New file:  {NEW_PATH}  ({COMMON_COUNT + NEW_ONLY_COUNT:,} rows)")
    print(f"Common:    {COMMON_COUNT:,}")
    print(f"Removed:   {OLD_ONLY_COUNT:,}")
    print(f"New:       {NEW_ONLY_COUNT:,}")
    print(f"Modified:  {MODIFIED_COUNT:,}")


if __name__ == "__main__":
    main()
