Convert text prompts to sql for your postgres db.

Grabs database DDL to use in the GPT prompts, creating working queries.

To run:
`
 go run aisql.go -K [openai auth key] -C '[postgres connection string]'
`

Example:
```
-> Give me all animals who's name starts with D
Response start=======
SELECT * 
FROM animals 
WHERE name LIKE 'D%' -- This query will return all animals whose name starts with the letter D. The LIKE operator is used to search for a pattern in a column. The '%' wildcard character is used to match any number of characters after the letter D.
Response finish======
Execute query? [y/N]
y
 id  |   name    |   type   | age |      registered_at       |              image               | owner_id 
-----+-----------+----------+-----+--------------------------+----------------------------------+----------
  54 | Devon Rex | insect   |  14 | 2021-06-18T20:40:34.991Z | http://placeimg.com/640/480/cats |       17 
 104 | Devon Rex | cetacean |   4 | 2021-11-02T06:34:56.853Z | http://placeimg.com/640/480/cats |       37 
(2 rows)
-> 
```