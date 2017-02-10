import expect from 'expect';

import { licenseSuccess } from 'redux/nodes/auth/actions';
import { licenseStub } from 'test/stubs';
import { reduxMockStore } from 'test/helpers';

const validLicense = licenseStub();

describe('nag_message - middleware', () => {
  it('dispatches a persistent flash message when a license is overused', () => {
    const mockStore = reduxMockStore();
    const overusedLicense = {
      ...validLicense,
      allowed_hosts: 2,
      hosts: 3,
    };
    const expectedNagMessageAction = {
      type: 'SHOW_PERSISTENT_FLASH',
      payload: {
        message: 'Please upgrade your Kolide license',
      },
    };

    mockStore.dispatch(licenseSuccess(overusedLicense));
    expect(mockStore.getActions()).toInclude(expectedNagMessageAction);
  });

  it('dispatches a persistent flash message when a license is expired', () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);

    const mockStore = reduxMockStore();

    const expiredLicense = { ...validLicense, expiry: yesterday.toISOString() };
    const expectedNagMessageAction = {
      type: 'SHOW_PERSISTENT_FLASH',
      payload: {
        message: 'Please upgrade your Kolide license',
      },
    };

    mockStore.dispatch(licenseSuccess(expiredLicense));
    expect(mockStore.getActions()).toInclude(expectedNagMessageAction);
  });

  it('dispatches a persistent flash message when a license is revoked', () => {
    const mockStore = reduxMockStore();

    const revokedLicense = { ...validLicense, revoked: true };
    const expectedNagMessageAction = {
      type: 'SHOW_PERSISTENT_FLASH',
      payload: {
        message: 'Please upgrade your Kolide license',
      },
    };

    mockStore.dispatch(licenseSuccess(revokedLicense));
    expect(mockStore.getActions()).toInclude(expectedNagMessageAction);
  });

  it('dispatches an action to clear persistent flash message when the license is valid', () => {
    const mockStore = reduxMockStore();
    const expectedNagMessageAction = { type: 'HIDE_PERSISTENT_FLASH' };

    mockStore.dispatch(licenseSuccess(validLicense));
    expect(mockStore.getActions()).toInclude(expectedNagMessageAction);
  });
});

