# Backend patterns

The backend software patterns that we follow in Fleet.

> NOTE: There are always exceptions to the rules, but we try to follow these patterns as much as possible unless a specific use case calls
> for something else. These should be discussed within the team and documented before merging.

## MySQL

Use high precision for all time fields. Precise timestamps make sure that we can accurately track when records were created and updated,
keep records in order with a reliable sort, and speed up testing by not having to wait for the time to
update. [MySQL reference](https://dev.mysql.com/doc/refman/8.4/en/date-and-time-type-syntax.html). [Backend sync where discussed](https://us-65885.app.gong.io/call?id=8041045095900447703).
Example:

```sql
CREATE TABLE `sample` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
);
```

Do not use [goqu](https://github.com/doug-martin/goqu); use MySQL queries directly. Searching for, understanding, and debugging direct MySQL
queries is easier. If needing to modify an existing `goqu` query, try to rewrite it in
MySQL. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=8041045095900447703).

### Data retention

Sometimes we need data from rows that have been deleted from DB. For example, the activity feed may be retained forever, and it needs user info (or host info) that may not exist anymore.
Going forward, we need to keep this data in a dedicated table(s). A reference unmerged PR is [here](https://github.com/fleetdm/fleet/pull/17472/files#diff-57a635e42320a87dd15a3ae03d66834f2cbc4fcdb5f3ebb7075d966b96f760afR16).
The `id` may be the same as that of the original table. For example, if the `user` row is deleted, a
new entry with the same `user.id` can be added to `user_persistent_info`.

## API Patterns

These are a collection of API patterns that we follow. They are not set in stone
but should be considered the norm where we only deviate from when there is a valid
reason to do so.

### Resource IDs

Resource ids should always be returned as a number.

```jsonc
// response for GET /queries/1

// good
{
  "query": {
    "id": 1
    ...
  }
}

// bad
{
  "query": {
    "id": "1"
    ...
  }
}
```

### Resourse response

The response for a resource should always be a JSON object with the resource
as the key. This allows for extending the resposne JSON in the future if needed.

```jsonc
// response for GET /queries/1

// good
{
  "query": {
    "id": 1,
    "name": "Test Query",
    ...
  }
}

// bad
{
  "id": 1,
  "name": "Test Query",
  ...
}

// response for GET /queries

// good
{
  "queries": [
    {
      "id": 1,
      ...
    },
    {
      "id": 2,
      ...
    }
  ]
}

// bad
[
  {
    "id": 1,
    ...
  },
  {
    "id": 2,
    ...
  }
]
```

## Pagination for list items endpoint

Endpoints that return a collection of resources should have pagination `meta` data in the
response to let the client know if there are more results that they can request.

```jsonc
// response to GET /software/titles endpoint

// good
{
  "software_titles": [{...}],
  "meta": {
    "has_next_results": true,
    "has_previous_results": false
  }
}

// bad
{
  "software_titles": [{...}]
}
```

## Empty values in response

We have a few conventions for empty values in a response. They are:

- string values return an empty string
- numbers and objects return `null`
- arrays return an empty array
- we do not omit response values

we do this so it is clearer to the client what data they have to work with in the API response.

```jsonc
// response from GET /labels/1

// good
{
  "label": {
    "id": 1,
    "description": ""
    ...
  }
}

// bad
{
  "label": {
    "id": 1,
    ... // "description" value is omitted
  }
}
```

```jsonc
// response from GET /config

// good
{
  ...
  "mdm": {
    "macos_settings": {
      "custom_settings": []
    }
  }
  ...
}

// bad
{
  ...
  "mdm": {
    "macos_settings": {
      "custom_settings": null // "custom_settings" set to null
    }
  }
  ...
}

// bad
{
  ...
  "mdm": {
    "macos_settings": {} // "custom_settings" has been omitted
  }
  ...
}
```
