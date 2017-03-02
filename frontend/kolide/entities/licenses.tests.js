import expect from 'expect';
import nock from 'nock';

import Kolide from 'kolide';
import { licenseStub } from 'test/stubs';
import mocks from 'test/mocks';

const { licenses: licenseMocks } = mocks;

describe('Kolide - API client (licenses)', () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = 'valid-bearer-token';
  const validLicense = licenseStub();

  describe('#create', () => {
    it('calls the correct endpoint with the correct parameters', () => {
      const jwtToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ';
      const request = licenseMocks.create.valid(bearerToken, jwtToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.license.create(jwtToken)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });

    it('changes 0 allowed_hosts to Unlimited', () => {
      const jwtToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ';
      const unlimitedHosts = { ...validLicense, allowed_hosts: 0 };

      licenseMocks.create.valid(bearerToken, jwtToken, unlimitedHosts);

      Kolide.setBearerToken(bearerToken);
      return Kolide.license.create(jwtToken)
        .then((response) => {
          expect(response.allowed_hosts).toEqual('Unlimited', 'Expected there to be unlimited allowed hosts');
        });
    });
  });

  describe('#load', () => {
    it('calls the correct endpoint with the correct parameters', () => {
      const request = licenseMocks.load.valid(bearerToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.license.load()
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });

    it('changes 0 allowed_hosts to Unlimited', () => {
      const unlimitedHosts = { ...validLicense, allowed_hosts: 0 };

      licenseMocks.load.valid(bearerToken, unlimitedHosts);

      Kolide.setBearerToken(bearerToken);
      return Kolide.license.load()
        .then((response) => {
          expect(response.allowed_hosts).toEqual('Unlimited', 'Expected there to be unlimited allowed hosts');
        });
    });
  });

  describe('#setup', () => {
    it('calls the correct endpoint with the correct parameters', () => {
      const jwtToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ';
      const request = licenseMocks.setup.valid(bearerToken, jwtToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.license.setup(jwtToken)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });

    it('changes 0 allowed_hosts to Unlimited', () => {
      const jwtToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ';
      const unlimitedHosts = { ...validLicense, allowed_hosts: 0 };

      licenseMocks.setup.valid(bearerToken, jwtToken, unlimitedHosts);

      Kolide.setBearerToken(bearerToken);
      return Kolide.license.setup(jwtToken)
        .then((response) => {
          expect(response.allowed_hosts).toEqual('Unlimited', 'Expected there to be unlimited allowed hosts');
        });
    });
  });
});

