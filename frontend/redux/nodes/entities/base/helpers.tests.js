import expect from 'expect';

import { entitiesExceptID } from './helpers';

describe('reduxConfig - helpers', () => {
  describe('#entitiesExceptID', () => {
    it('returns an empty object if all ids are deleted', () => {
      const entities = {
        1: { name: 'Gnar' },
      };
      const id = 1;

      expect(entitiesExceptID(entities, id)).toEqual({});
    });

    it('removes the object with the key of the specified id', () => {
      const entities = {
        1: { name: 'Gnar' },
        2: { name: 'Dog' },
      };
      const id = 1;

      expect(entitiesExceptID(entities, id)).toEqual({
        2: { name: 'Dog' },
      });
    });
  });
});
