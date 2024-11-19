# Query Analysis Report

**Date of Analysis**: 2024-01-01 01:02:03  
**Analyzed File**: `../testdata/bigger_slow_query_log.json`

## Top Queries
|Query ID|Usage Count|Total Query Time (ms)|Avg Query Time (ms)|Total Rows Examined|
|---|---|---|---|---|
|Q1|3|0.61|0.20|30000|
|Q2|3|0.58|0.19|17000|
|Q3|2|0.49|0.25|16000|
|Q4|2|0.40|0.20|20000|
|Q5|2|0.37|0.19|16000|
|Q6|2|0.37|0.19|16000|
|Q7|2|0.34|0.17|8500|
|Q8|2|0.33|0.17|6000|
|Q9|2|0.31|0.16|15000|
|Q10|1|0.22|0.22|8000|

### Query Details
#### Q1
```sql
SELECT `m`.`sender_id`, COUNT(DISTINCT `m`.`receiver_id`) AS `unique_receivers` FROM `messages` AS `m` GROUP BY `m`.`sender_id` HAVING COUNT(DISTINCT `m`.`receiver_id`) > :_unique_receivers /* INT64 */
```

#### Q2
```sql
SELECT `u`.`username`, sum(`o`.`total_amount`) AS `total_spent` FROM `users` AS `u` JOIN `orders` AS `o` ON `u`.`id` = `o`.`user_id` WHERE `o`.`created_at` BETWEEN :1 /* VARCHAR */ AND :2 /* VARCHAR */ GROUP BY `u`.`id` HAVING sum(`o`.`total_amount`) > :_total_spent /* INT64 */
```

#### Q3
```sql
SELECT `u`.`id`, `u`.`username` FROM `users` AS `u` LEFT JOIN `orders` AS `o` ON `u`.`id` = `o`.`user_id` WHERE `o`.`id` IS NULL
```

#### Q4
```sql
SELECT `c`.`name`, sum(`oi`.`price` * `oi`.`quantity`) AS `total_sales` FROM `categories` AS `c` JOIN `products` AS `p` ON `c`.`id` = `p`.`category_id` JOIN `order_items` AS `oi` ON `p`.`id` = `oi`.`product_id` GROUP BY `c`.`id` ORDER BY sum(`oi`.`price` * `oi`.`quantity`) DESC LIMIT :1 /* INT64 */
```

#### Q5
```sql
SELECT `c`.`name`, COUNT(`o`.`id`) AS `order_count` FROM `categories` AS `c` JOIN `products` AS `p` ON `c`.`id` = `p`.`category_id` JOIN `order_items` AS `oi` ON `p`.`id` = `oi`.`product_id` JOIN `orders` AS `o` ON `oi`.`order_id` = `o`.`id` GROUP BY `c`.`id`
```

#### Q6
```sql
SELECT DATE(`o`.`created_at`) AS `order_date`, count(*) AS `order_count` FROM `orders` AS `o` WHERE `o`.`created_at` >= DATE_SUB(now(), INTERVAL :1 /* INT64 */ day) GROUP BY DATE(`o`.`created_at`)
```

#### Q7
```sql
SELECT `o`.`id`, `o`.`created_at` FROM `orders` AS `o` LEFT JOIN `shipments` AS `s` ON `o`.`id` = `s`.`order_id` WHERE `s`.`shipped_date` IS NULL AND `o`.`created_at` < DATE_SUB(now(), INTERVAL :1 /* INT64 */ day)
```

#### Q8
```sql
SELECT `p`.`payment_method`, avg(`o`.`total_amount`) AS `avg_order_value` FROM `payments` AS `p` JOIN `orders` AS `o` ON `p`.`order_id` = `o`.`id` GROUP BY `p`.`payment_method`
```

#### Q9
```sql
SELECT `p`.`name`, `i`.`stock_level` FROM `products` AS `p` JOIN `inventory` AS `i` ON `p`.`id` = `i`.`product_id` WHERE `i`.`stock_level` < :_i_stock_level /* INT64 */
```

#### Q10
```sql
SELECT `u`.`id`, `u`.`username` FROM `users` AS `u` JOIN `orders` AS `o` ON `u`.`id` = `o`.`user_id` JOIN `reviews` AS `r` ON `u`.`id` = `r`.`user_id` WHERE `o`.`created_at` >= DATE_SUB(now(), INTERVAL :1 /* INT64 */ month) AND `r`.`created_at` >= DATE_SUB(now(), INTERVAL :1 /* INT64 */ month)
```

## Tables
|Table Name|Reads|Writes|
|---|---|---|
|orders|14|0|
|products|10|0|
|users|6|0|
|categories|4|0|
|order_items|4|0|
|reviews|4|0|
|inventory|3|0|
|messages|3|0|
|payments|2|0|
|shipments|2|0|

### Column Usage
#### Table: `orders` (14 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|created_at|WHERE RANGE|57%|
|id|JOIN|43%|
|user_id|JOIN|43%|

#### Table: `products` (10 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|id|JOIN|100%|
||GROUP|30%|
|category_id|JOIN|40%|

#### Table: `users` (6 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|id|JOIN|100%|
||GROUP|50%|

#### Table: `categories` (4 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|id|JOIN|100%|
||GROUP|100%|

#### Table: `order_items` (4 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|product_id|JOIN|100%|
|order_id|JOIN|50%|

#### Table: `reviews` (4 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|product_id|JOIN|75%|
|created_at|WHERE RANGE|50%|
|user_id|JOIN|25%|

#### Table: `inventory` (3 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|product_id|JOIN|100%|
|stock_level|WHERE RANGE|100%|

#### Table: `messages` (3 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|sender_id|GROUP|100%|

#### Table: `payments` (2 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|order_id|JOIN|100%|
|payment_method|GROUP|100%|

#### Table: `shipments` (2 reads and 0 writes)
|Column|Position|Used %|
|---|---|---|
|order_id|JOIN|100%|

## Tables Joined
```
orders ↔ users (Occurrences: 3)
└─ orders.user_id = users.id

categories ↔ products (Occurrences: 2)
└─ categories.id = products.category_id

inventory ↔ products (Occurrences: 2)
└─ inventory.product_id = products.id

order_items ↔ products (Occurrences: 2)
└─ order_items.product_id = products.id

products ↔ reviews (Occurrences: 2)
└─ products.id = reviews.product_id

order_items ↔ orders (Occurrences: 1)
└─ order_items.order_id = orders.id

orders ↔ payments (Occurrences: 1)
└─ orders.id = payments.order_id

orders ↔ shipments (Occurrences: 1)
└─ orders.id = shipments.order_id

reviews ↔ users (Occurrences: 1)
└─ reviews.user_id = users.id

```
## Failures
|Error|Count|
|---|---|
|syntax error at position 2|1|
|syntax error at position 14 near 'timestamp'|1|

