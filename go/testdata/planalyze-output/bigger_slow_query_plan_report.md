# Query Planning Report

**Date of Analysis**: 2024-01-01 01:02:03  
**Analyzed File**: ../testdata/keys-output/bigger_slow_query_log.json

|Plan Complexity|Count|
|---|---|
|Pass-through|0|
|Simple routed|2|
|Complex routed|11|
|Unplannable|0|
|Total|13|


# Simple routed Queries

## Query

```sql
SELECT `p`.`name`, `i`.`stock_level` FROM `products` AS `p` JOIN `inventory` AS `i` ON `p`.`id` = `i`.`product_id` WHERE `i`.`stock_level` < :_i_stock_level /* INT64 */
```

## Plan

```json
{
  "OperatorType": "Route",
  "Variant": "Scatter",
  "Keyspace": {
    "Name": "main",
    "Sharded": true
  },
  "FieldQuery": "select p.`name`, i.stock_level from products as p, inventory as i where 1 != 1",
  "Query": "select p.`name`, i.stock_level from products as p, inventory as i where i.stock_level < :_i_stock_level and p.id = i.product_id",
  "Table": "inventory, products"
}

```


## Query

```sql
SELECT `p`.`name`, `i`.`stock_level` FROM `products` AS `p` JOIN `inventory` AS `i` ON `p`.`id` = `i`.`product_id` WHERE `i`.`stock_level` BETWEEN :1 /* INT64 */ AND :2 /* INT64 */
```

## Plan

```json
{
  "OperatorType": "Route",
  "Variant": "Scatter",
  "Keyspace": {
    "Name": "main",
    "Sharded": true
  },
  "FieldQuery": "select p.`name`, i.stock_level from products as p, inventory as i where 1 != 1",
  "Query": "select p.`name`, i.stock_level from products as p, inventory as i where i.stock_level between :1 and :2 and p.id = i.product_id",
  "Table": "inventory, products"
}

```


# Complex routed Queries

## Query

```sql
SELECT `p`.`name`, avg(`r`.`rating`) AS `avg_rating` FROM `products` AS `p` JOIN `reviews` AS `r` ON `p`.`id` = `r`.`product_id` GROUP BY `p`.`id` ORDER BY avg(`r`.`rating`) DESC LIMIT :1 /* INT64 */
```

## Plan

```json
{
  "OperatorType": "Limit",
  "Count": "_vt_column_1",
  "Inputs": [
    {
      "OperatorType": "Route",
      "Variant": "Scatter",
      "Keyspace": {
        "Name": "main",
        "Sharded": true
      },
      "FieldQuery": "select p.`name`, avg(r.rating) as avg_rating from products as p, reviews as r where 1 != 1 group by p.id",
      "OrderBy": "1 DESC COLLATE utf8mb4_0900_ai_ci",
      "Query": "select p.`name`, avg(r.rating) as avg_rating from products as p, reviews as r where p.id = r.product_id group by p.id order by avg(r.rating) desc limit :1",
      "Table": "products, reviews"
    }
  ]
}

```


## Query

```sql
SELECT `u`.`username`, sum(`o`.`total_amount`) AS `total_spent` FROM `users` AS `u` JOIN `orders` AS `o` ON `u`.`id` = `o`.`user_id` WHERE `o`.`created_at` BETWEEN :1 /* VARCHAR */ AND :2 /* VARCHAR */ GROUP BY `u`.`id` HAVING sum(`o`.`total_amount`) > :_total_spent /* INT64 */
```

## Plan

```json
{
  "OperatorType": "Filter",
  "Predicate": "sum(o.total_amount) > :_total_spent",
  "ResultColumns": 2,
  "Inputs": [
    {
      "OperatorType": "Aggregate",
      "Variant": "Ordered",
      "Aggregates": "any_value(0) AS username, sum(1) AS total_spent",
      "GroupBy": "(2|3)",
      "Inputs": [
        {
          "OperatorType": "Projection",
          "Expressions": [
            ":0 as username",
            "sum(o.total_amount) * count(*) as total_spent",
            ":3 as id",
            ":4 as weight_string(u.id)"
          ],
          "Inputs": [
            {
              "OperatorType": "Sort",
              "Variant": "Memory",
              "OrderBy": "(3|4) ASC",
              "Inputs": [
                {
                  "OperatorType": "Join",
                  "Variant": "Join",
                  "JoinColumnIndexes": "R:0,L:0,R:1,R:2,R:3",
                  "JoinVars": {
                    "o_user_id": 1
                  },
                  "TableName": "orders_users",
                  "Inputs": [
                    {
                      "OperatorType": "Route",
                      "Variant": "Scatter",
                      "Keyspace": {
                        "Name": "main",
                        "Sharded": true
                      },
                      "FieldQuery": "select sum(o.total_amount) as total_spent, o.user_id from orders as o where 1 != 1 group by o.user_id",
                      "Query": "select sum(o.total_amount) as total_spent, o.user_id from orders as o where o.created_at between :1 and :2 group by o.user_id",
                      "Table": "orders"
                    },
                    {
                      "OperatorType": "Route",
                      "Variant": "EqualUnique",
                      "Keyspace": {
                        "Name": "main",
                        "Sharded": true
                      },
                      "FieldQuery": "select u.username, count(*), u.id, weight_string(u.id) from users as u where 1 != 1 group by u.id, weight_string(u.id)",
                      "Query": "select u.username, count(*), u.id, weight_string(u.id) from users as u where u.id = :o_user_id group by u.id, weight_string(u.id)",
                      "Table": "users",
                      "Values": [
                        ":o_user_id"
                      ],
                      "Vindex": "xxhash"
                    }
                  ]
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT `c`.`name`, COUNT(`o`.`id`) AS `order_count` FROM `categories` AS `c` JOIN `products` AS `p` ON `c`.`id` = `p`.`category_id` JOIN `order_items` AS `oi` ON `p`.`id` = `oi`.`product_id` JOIN `orders` AS `o` ON `oi`.`order_id` = `o`.`id` GROUP BY `c`.`id`
```

## Plan

```json
{
  "OperatorType": "Aggregate",
  "Variant": "Ordered",
  "Aggregates": "any_value(0) AS name, sum_count(1) AS order_count",
  "GroupBy": "(2|3)",
  "ResultColumns": 2,
  "Inputs": [
    {
      "OperatorType": "Projection",
      "Expressions": [
        ":0 as name",
        "count(o.id) * count(*) as order_count",
        ":3 as id",
        ":4 as weight_string(c.id)"
      ],
      "Inputs": [
        {
          "OperatorType": "Sort",
          "Variant": "Memory",
          "OrderBy": "(3|4) ASC",
          "Inputs": [
            {
              "OperatorType": "Join",
              "Variant": "Join",
              "JoinColumnIndexes": "R:0,L:0,R:1,R:2,R:3",
              "JoinVars": {
                "oi_product_id": 1
              },
              "TableName": "order_items_orders_products_categories",
              "Inputs": [
                {
                  "OperatorType": "Projection",
                  "Expressions": [
                    "count(*) * count(o.id) as order_count",
                    ":2 as product_id"
                  ],
                  "Inputs": [
                    {
                      "OperatorType": "Join",
                      "Variant": "Join",
                      "JoinColumnIndexes": "R:0,L:0,L:1",
                      "JoinVars": {
                        "oi_order_id": 2
                      },
                      "TableName": "order_items_orders",
                      "Inputs": [
                        {
                          "OperatorType": "Route",
                          "Variant": "Scatter",
                          "Keyspace": {
                            "Name": "main",
                            "Sharded": true
                          },
                          "FieldQuery": "select count(*), oi.product_id, oi.order_id from order_items as oi where 1 != 1 group by oi.product_id, oi.order_id",
                          "Query": "select count(*), oi.product_id, oi.order_id from order_items as oi group by oi.product_id, oi.order_id",
                          "Table": "order_items"
                        },
                        {
                          "OperatorType": "Route",
                          "Variant": "EqualUnique",
                          "Keyspace": {
                            "Name": "main",
                            "Sharded": true
                          },
                          "FieldQuery": "select count(o.id) as order_count from orders as o where 1 != 1 group by .0",
                          "Query": "select count(o.id) as order_count from orders as o where o.id = :oi_order_id group by .0",
                          "Table": "orders",
                          "Values": [
                            ":oi_order_id"
                          ],
                          "Vindex": "xxhash"
                        }
                      ]
                    }
                  ]
                },
                {
                  "OperatorType": "Projection",
                  "Expressions": [
                    ":0 as name",
                    "count(*) * count(*) as count(*)",
                    ":3 as id",
                    ":4 as weight_string(c.id)"
                  ],
                  "Inputs": [
                    {
                      "OperatorType": "Join",
                      "Variant": "Join",
                      "JoinColumnIndexes": "R:0,L:0,R:1,R:2,R:3",
                      "JoinVars": {
                        "p_category_id": 1
                      },
                      "TableName": "products_categories",
                      "Inputs": [
                        {
                          "OperatorType": "Route",
                          "Variant": "EqualUnique",
                          "Keyspace": {
                            "Name": "main",
                            "Sharded": true
                          },
                          "FieldQuery": "select count(*), p.category_id from products as p where 1 != 1 group by p.category_id",
                          "Query": "select count(*), p.category_id from products as p where p.id = :oi_product_id group by p.category_id",
                          "Table": "products",
                          "Values": [
                            ":oi_product_id"
                          ],
                          "Vindex": "xxhash"
                        },
                        {
                          "OperatorType": "Route",
                          "Variant": "EqualUnique",
                          "Keyspace": {
                            "Name": "main",
                            "Sharded": true
                          },
                          "FieldQuery": "select c.`name`, count(*), c.id, weight_string(c.id) from categories as c where 1 != 1 group by c.id, weight_string(c.id)",
                          "Query": "select c.`name`, count(*), c.id, weight_string(c.id) from categories as c where c.id = :p_category_id group by c.id, weight_string(c.id)",
                          "Table": "categories",
                          "Values": [
                            ":p_category_id"
                          ],
                          "Vindex": "xxhash"
                        }
                      ]
                    }
                  ]
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT `c`.`name`, sum(`oi`.`price` * `oi`.`quantity`) AS `total_sales` FROM `categories` AS `c` JOIN `products` AS `p` ON `c`.`id` = `p`.`category_id` JOIN `order_items` AS `oi` ON `p`.`id` = `oi`.`product_id` GROUP BY `c`.`id` ORDER BY sum(`oi`.`price` * `oi`.`quantity`) DESC LIMIT :1 /* INT64 */
```

## Plan

```json
{
  "OperatorType": "Limit",
  "Count": "_vt_column_1",
  "Inputs": [
    {
      "OperatorType": "Sort",
      "Variant": "Memory",
      "OrderBy": "1 DESC COLLATE utf8mb4_0900_ai_ci",
      "ResultColumns": 2,
      "Inputs": [
        {
          "OperatorType": "Aggregate",
          "Variant": "Ordered",
          "Aggregates": "any_value(0) AS name, sum(1) AS total_sales",
          "GroupBy": "(2|3)",
          "Inputs": [
            {
              "OperatorType": "Projection",
              "Expressions": [
                ":0 as name",
                "sum(oi.price * oi.quantity) * count(*) as total_sales",
                ":3 as id",
                ":4 as weight_string(c.id)"
              ],
              "Inputs": [
                {
                  "OperatorType": "Sort",
                  "Variant": "Memory",
                  "OrderBy": "(3|4) ASC",
                  "Inputs": [
                    {
                      "OperatorType": "Join",
                      "Variant": "Join",
                      "JoinColumnIndexes": "R:0,L:0,R:1,R:2,R:3",
                      "JoinVars": {
                        "oi_product_id": 1
                      },
                      "TableName": "order_items_products_categories",
                      "Inputs": [
                        {
                          "OperatorType": "Route",
                          "Variant": "Scatter",
                          "Keyspace": {
                            "Name": "main",
                            "Sharded": true
                          },
                          "FieldQuery": "select sum(oi.price * oi.quantity) as total_sales, oi.product_id from order_items as oi where 1 != 1 group by oi.product_id",
                          "Query": "select sum(oi.price * oi.quantity) as total_sales, oi.product_id from order_items as oi group by oi.product_id",
                          "Table": "order_items"
                        },
                        {
                          "OperatorType": "Projection",
                          "Expressions": [
                            ":0 as name",
                            "count(*) * count(*) as count(*)",
                            ":3 as id",
                            ":4 as weight_string(c.id)"
                          ],
                          "Inputs": [
                            {
                              "OperatorType": "Join",
                              "Variant": "Join",
                              "JoinColumnIndexes": "R:0,L:0,R:1,R:2,R:3",
                              "JoinVars": {
                                "p_category_id": 1
                              },
                              "TableName": "products_categories",
                              "Inputs": [
                                {
                                  "OperatorType": "Route",
                                  "Variant": "EqualUnique",
                                  "Keyspace": {
                                    "Name": "main",
                                    "Sharded": true
                                  },
                                  "FieldQuery": "select count(*), p.category_id from products as p where 1 != 1 group by p.category_id",
                                  "Query": "select count(*), p.category_id from products as p where p.id = :oi_product_id group by p.category_id",
                                  "Table": "products",
                                  "Values": [
                                    ":oi_product_id"
                                  ],
                                  "Vindex": "xxhash"
                                },
                                {
                                  "OperatorType": "Route",
                                  "Variant": "EqualUnique",
                                  "Keyspace": {
                                    "Name": "main",
                                    "Sharded": true
                                  },
                                  "FieldQuery": "select c.`name`, count(*), c.id, weight_string(c.id) from categories as c where 1 != 1 group by c.id, weight_string(c.id)",
                                  "Query": "select c.`name`, count(*), c.id, weight_string(c.id) from categories as c where c.id = :p_category_id group by c.id, weight_string(c.id)",
                                  "Table": "categories",
                                  "Values": [
                                    ":p_category_id"
                                  ],
                                  "Vindex": "xxhash"
                                }
                              ]
                            }
                          ]
                        }
                      ]
                    }
                  ]
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT `o`.`id`, `o`.`created_at` FROM `orders` AS `o` LEFT JOIN `shipments` AS `s` ON `o`.`id` = `s`.`order_id` WHERE `s`.`shipped_date` IS NULL AND `o`.`created_at` < DATE_SUB(now(), INTERVAL :1 /* INT64 */ day)
```

## Plan

```json
{
  "OperatorType": "Filter",
  "Predicate": "s.shipped_date is null",
  "ResultColumns": 2,
  "Inputs": [
    {
      "OperatorType": "Join",
      "Variant": "LeftJoin",
      "JoinColumnIndexes": "L:0,L:1,R:0",
      "JoinVars": {
        "o_id": 0
      },
      "TableName": "orders_shipments",
      "Inputs": [
        {
          "OperatorType": "Route",
          "Variant": "Scatter",
          "Keyspace": {
            "Name": "main",
            "Sharded": true
          },
          "FieldQuery": "select o.id, o.created_at from orders as o where 1 != 1",
          "Query": "select o.id, o.created_at from orders as o where o.created_at < date_sub(now(), interval :1 day)",
          "Table": "orders"
        },
        {
          "OperatorType": "Route",
          "Variant": "Scatter",
          "Keyspace": {
            "Name": "main",
            "Sharded": true
          },
          "FieldQuery": "select s.shipped_date from shipments as s where 1 != 1",
          "Query": "select s.shipped_date from shipments as s where s.order_id = :o_id",
          "Table": "shipments"
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT `p`.`payment_method`, avg(`o`.`total_amount`) AS `avg_order_value` FROM `payments` AS `p` JOIN `orders` AS `o` ON `p`.`order_id` = `o`.`id` GROUP BY `p`.`payment_method`
```

## Plan

```json
{
  "OperatorType": "Projection",
  "Expressions": [
    ":0 as payment_method",
    "sum(o.total_amount) / count(o.total_amount) as avg_order_value"
  ],
  "Inputs": [
    {
      "OperatorType": "Aggregate",
      "Variant": "Ordered",
      "Aggregates": "sum(1) AS avg_order_value, sum_count(2) AS count(o.total_amount)",
      "GroupBy": "(0|3)",
      "Inputs": [
        {
          "OperatorType": "Projection",
          "Expressions": [
            ":3 as payment_method",
            "count(*) * sum(o.total_amount) as avg_order_value",
            "count(*) * count(o.total_amount) as count(o.total_amount)",
            ":4 as weight_string(p.payment_method)"
          ],
          "Inputs": [
            {
              "OperatorType": "Join",
              "Variant": "Join",
              "JoinColumnIndexes": "R:0,L:0,R:1,L:1,L:3",
              "JoinVars": {
                "p_order_id": 2
              },
              "TableName": "payments_orders",
              "Inputs": [
                {
                  "OperatorType": "Route",
                  "Variant": "Scatter",
                  "Keyspace": {
                    "Name": "main",
                    "Sharded": true
                  },
                  "FieldQuery": "select count(*), p.payment_method, p.order_id, weight_string(p.payment_method) from payments as p where 1 != 1 group by p.payment_method, p.order_id, weight_string(p.payment_method)",
                  "OrderBy": "(1|3) ASC",
                  "Query": "select count(*), p.payment_method, p.order_id, weight_string(p.payment_method) from payments as p group by p.payment_method, p.order_id, weight_string(p.payment_method) order by p.payment_method asc",
                  "Table": "payments"
                },
                {
                  "OperatorType": "Route",
                  "Variant": "EqualUnique",
                  "Keyspace": {
                    "Name": "main",
                    "Sharded": true
                  },
                  "FieldQuery": "select sum(o.total_amount) as avg_order_value, count(o.total_amount) from orders as o where 1 != 1 group by .0",
                  "Query": "select sum(o.total_amount) as avg_order_value, count(o.total_amount) from orders as o where o.id = :p_order_id group by .0",
                  "Table": "orders",
                  "Values": [
                    ":p_order_id"
                  ],
                  "Vindex": "xxhash"
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT DATE(`o`.`created_at`) AS `order_date`, count(*) AS `order_count` FROM `orders` AS `o` WHERE `o`.`created_at` >= DATE_SUB(now(), INTERVAL :1 /* INT64 */ day) GROUP BY DATE(`o`.`created_at`)
```

## Plan

```json
{
  "OperatorType": "Aggregate",
  "Variant": "Ordered",
  "Aggregates": "sum_count_star(1) AS order_count",
  "GroupBy": "(0|2)",
  "ResultColumns": 2,
  "Inputs": [
    {
      "OperatorType": "Route",
      "Variant": "Scatter",
      "Keyspace": {
        "Name": "main",
        "Sharded": true
      },
      "FieldQuery": "select DATE(o.created_at) as order_date, count(*) as order_count, weight_string(DATE(o.created_at)) from orders as o where 1 != 1 group by DATE(o.created_at), weight_string(DATE(o.created_at))",
      "OrderBy": "(0|2) ASC",
      "Query": "select DATE(o.created_at) as order_date, count(*) as order_count, weight_string(DATE(o.created_at)) from orders as o where o.created_at >= date_sub(now(), interval :1 day) group by DATE(o.created_at), weight_string(DATE(o.created_at)) order by DATE(o.created_at) asc",
      "Table": "orders"
    }
  ]
}

```


## Query

```sql
SELECT `m`.`sender_id`, COUNT(DISTINCT `m`.`receiver_id`) AS `unique_receivers` FROM `messages` AS `m` GROUP BY `m`.`sender_id` HAVING COUNT(DISTINCT `m`.`receiver_id`) > :_unique_receivers /* INT64 */
```

## Plan

```json
{
  "OperatorType": "Filter",
  "Predicate": "count(distinct m.receiver_id) > :_unique_receivers",
  "ResultColumns": 2,
  "Inputs": [
    {
      "OperatorType": "Aggregate",
      "Variant": "Ordered",
      "Aggregates": "count_distinct(1|3) AS unique_receivers",
      "GroupBy": "(0|2)",
      "Inputs": [
        {
          "OperatorType": "Route",
          "Variant": "Scatter",
          "Keyspace": {
            "Name": "main",
            "Sharded": true
          },
          "FieldQuery": "select m.sender_id, m.receiver_id, weight_string(m.sender_id), weight_string(m.receiver_id) from messages as m where 1 != 1 group by m.sender_id, m.receiver_id, weight_string(m.sender_id), weight_string(m.receiver_id)",
          "OrderBy": "(0|2) ASC, (1|3) ASC",
          "Query": "select m.sender_id, m.receiver_id, weight_string(m.sender_id), weight_string(m.receiver_id) from messages as m group by m.sender_id, m.receiver_id, weight_string(m.sender_id), weight_string(m.receiver_id) order by m.sender_id asc, m.receiver_id asc",
          "Table": "messages"
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT `u`.`id`, `u`.`username` FROM `users` AS `u` LEFT JOIN `orders` AS `o` ON `u`.`id` = `o`.`user_id` WHERE `o`.`id` IS NULL
```

## Plan

```json
{
  "OperatorType": "Filter",
  "Predicate": "o.id is null",
  "ResultColumns": 2,
  "Inputs": [
    {
      "OperatorType": "Join",
      "Variant": "LeftJoin",
      "JoinColumnIndexes": "L:0,L:1,R:0",
      "JoinVars": {
        "u_id": 0
      },
      "TableName": "users_orders",
      "Inputs": [
        {
          "OperatorType": "Route",
          "Variant": "Scatter",
          "Keyspace": {
            "Name": "main",
            "Sharded": true
          },
          "FieldQuery": "select u.id, u.username from users as u where 1 != 1",
          "Query": "select u.id, u.username from users as u",
          "Table": "users"
        },
        {
          "OperatorType": "Route",
          "Variant": "Scatter",
          "Keyspace": {
            "Name": "main",
            "Sharded": true
          },
          "FieldQuery": "select o.id from orders as o where 1 != 1",
          "Query": "select o.id from orders as o where o.user_id = :u_id",
          "Table": "orders"
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT `u`.`id`, `u`.`username` FROM `users` AS `u` JOIN `orders` AS `o` ON `u`.`id` = `o`.`user_id` JOIN `reviews` AS `r` ON `u`.`id` = `r`.`user_id` WHERE `o`.`created_at` >= DATE_SUB(now(), INTERVAL :1 /* INT64 */ month) AND `r`.`created_at` >= DATE_SUB(now(), INTERVAL :1 /* INT64 */ month)
```

## Plan

```json
{
  "OperatorType": "Join",
  "Variant": "Join",
  "JoinColumnIndexes": "R:0,R:1",
  "JoinVars": {
    "r_user_id": 0
  },
  "TableName": "reviews_orders_users",
  "Inputs": [
    {
      "OperatorType": "Route",
      "Variant": "Scatter",
      "Keyspace": {
        "Name": "main",
        "Sharded": true
      },
      "FieldQuery": "select r.user_id from reviews as r where 1 != 1",
      "Query": "select r.user_id from reviews as r where r.created_at >= date_sub(now(), interval :1 month)",
      "Table": "reviews"
    },
    {
      "OperatorType": "Join",
      "Variant": "Join",
      "JoinColumnIndexes": "R:0,R:1",
      "JoinVars": {
        "o_user_id": 0
      },
      "TableName": "orders_users",
      "Inputs": [
        {
          "OperatorType": "Route",
          "Variant": "Scatter",
          "Keyspace": {
            "Name": "main",
            "Sharded": true
          },
          "FieldQuery": "select o.user_id from orders as o where 1 != 1",
          "Query": "select o.user_id from orders as o where o.created_at >= date_sub(now(), interval :1 month)",
          "Table": "orders"
        },
        {
          "OperatorType": "Route",
          "Variant": "EqualUnique",
          "Keyspace": {
            "Name": "main",
            "Sharded": true
          },
          "FieldQuery": "select u.id, u.username from users as u where 1 != 1",
          "Query": "select u.id, u.username from users as u where u.id = :r_user_id and u.id = :o_user_id",
          "Table": "users",
          "Values": [
            ":r_user_id"
          ],
          "Vindex": "xxhash"
        }
      ]
    }
  ]
}

```


## Query

```sql
SELECT `p`.`name`, avg(`r`.`rating`) AS `avg_rating` FROM `products` AS `p` JOIN `reviews` AS `r` ON `p`.`id` = `r`.`product_id` WHERE `r`.`created_at` >= DATE_SUB(now(), INTERVAL :1 /* INT64 */ week) GROUP BY `p`.`id` ORDER BY avg(`r`.`rating`) DESC LIMIT :2 /* INT64 */
```

## Plan

```json
{
  "OperatorType": "Limit",
  "Count": "_vt_column_2",
  "Inputs": [
    {
      "OperatorType": "Route",
      "Variant": "Scatter",
      "Keyspace": {
        "Name": "main",
        "Sharded": true
      },
      "FieldQuery": "select p.`name`, avg(r.rating) as avg_rating from products as p, reviews as r where 1 != 1 group by p.id",
      "OrderBy": "1 DESC COLLATE utf8mb4_0900_ai_ci",
      "Query": "select p.`name`, avg(r.rating) as avg_rating from products as p, reviews as r where r.created_at >= date_sub(now(), interval :1 week) and p.id = r.product_id group by p.id order by avg(r.rating) desc limit :2",
      "Table": "products, reviews"
    }
  ]
}

```


