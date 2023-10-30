WITH human_accounts AS (
  SELECT username FROM users u WHERE u.uid >= 1000 AND u.uid < 60000
),
  user_xrdb_values AS (
    SELECT * FROM kolide_xrdb kx WHERE kx.username IN (SELECT username from human_accounts)
    )

SELECT * FROM user_xrdb_values;
