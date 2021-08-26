# Kolide API Client

The Kolide API Client is used for communicating with the Kolide API. Kolide has
a number of entities (hosts, labels, packs, queries, users, etc), all of which
have CRUD methods to perform on a specific entity or collection of entities.

Entities are assigned to the API Client in the `constructor` function. Each
entity's methods can be found in the [`/frontend/fleet/entities`](./entities) directory.

The CRUD methods that are typically implemented in the API client are as follows:

`create`

* The `create` method is used for creating a new entity. The input parameter is
typically an object containing the attributes for the new entity.

`destroy`

* The `destroy` method is used for deleting an entity. Then input parameter is
typically the entity to be deleted.

`load`

* The `load` method is used for loading a single entity. The input parameter is
typically the `id` of the entity to load.

`loadAll`

* The `loadAll` method is used for loading all of the entities.

`update`

* The `update` method is used to update an entity. The input parameters are
  typically the entity being updated and an object with the updated attributes.
