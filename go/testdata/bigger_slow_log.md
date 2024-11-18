# Query Analysis Report

**Date of Analysis**: 2024-01-01 01:02:03  
**Analyzed File**: `../testdata/bigger_slow_query_log.json`

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

