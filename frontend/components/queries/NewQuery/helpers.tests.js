import expect from 'expect';

import helpers from './helpers';

const malformedQuery = 'this is not a thing';
const validQuery = 'SELECT * FROM users';
const createQuery = 'CREATE TABLE users (LastName varchar(255))';
const insertQuery = 'INSERT INTO users (name) values ("Mike")';

describe('NewQuery - helpers', () => {
  describe('#validateQuery', () => {
    const { validateQuery } = helpers;

    it('rejects malformed queries', () => {
      const { error, valid } = validateQuery(malformedQuery);

      expect(valid).toEqual(false);
      expect(error).toEqual('Syntax error found near WITH Clause (Statement)');
    });

    it('rejects create queries', () => {
      const { error, valid } = validateQuery(createQuery);
      expect(valid).toEqual(false);
      expect(error).toEqual('Cannot INSERT or CREATE in osquery queries');
    });

    it('rejects insert queries', () => {
      const { error, valid } = validateQuery(insertQuery);
      expect(valid).toEqual(false);
      expect(error).toEqual('Cannot INSERT or CREATE in osquery queries');
    });

    it('accepts valid queries', () => {
      const { error, valid } = validateQuery(validQuery);
      expect(valid).toEqual(true);
      expect(error).toNotExist();
    });
  });
});
