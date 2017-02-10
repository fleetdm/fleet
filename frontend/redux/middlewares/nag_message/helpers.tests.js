import expect from 'expect';

import helpers from 'redux/middlewares/nag_message/helpers';
import { licenseStub } from 'test/stubs';

const validLicense = licenseStub();

describe('Nag message middleware - helpers', () => {
  describe('#shouldNagUser', () => {
    const { shouldNagUser } = helpers;

    it('returns true when there are more hosts than allowed hosts', () => {
      const overusedLicense = {
        ...validLicense,
        allowed_hosts: 2,
        hosts: 3,
      };

      expect(shouldNagUser({ license: overusedLicense })).toEqual(true);
    });

    it('returns true when the license is expired', () => {
      const yesterday = new Date();
      yesterday.setDate(yesterday.getDate() - 1);

      const expiredLicense = { ...validLicense, expiry: yesterday.toISOString() };

      expect(shouldNagUser({ license: expiredLicense })).toEqual(true, 'Expected the expired license to return true');
    });

    it('returns true when the license is revoked', () => {
      const revokedLicense = { ...validLicense, revoked: true };

      expect(shouldNagUser({ license: revokedLicense })).toEqual(true, 'Expected the revoked license to return true');
    });

    it('returns false when the license is valid', () => {
      expect(shouldNagUser({ license: validLicense })).toEqual(false, 'Expected the valid license to return false');
    });
  });
});
