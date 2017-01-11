import expect from 'expect';
import { omit } from 'lodash';

import { configStub } from 'test/stubs';
import helpers from 'kolide/helpers';

const label1 = { id: 1, target_type: 'labels' };
const label2 = { id: 2, target_type: 'labels' };
const host1 = { id: 6, target_type: 'hosts' };
const host2 = { id: 5, target_type: 'hosts' };

describe('Kolide API - helpers', () => {
  describe('#labelSlug', () => {
    it('creates a slug for the label', () => {
      expect(helpers.labelSlug({ display_text: 'All Hosts' })).toEqual('all-hosts');
      expect(helpers.labelSlug({ display_text: 'windows' })).toEqual('windows');
    });
  });

  describe('#formatConfigDataForServer', () => {
    const { formatConfigDataForServer } = helpers;
    const config = {
      org_name: 'Kolide',
      org_logo_url: '0.0.0.0:8080/logo.png',
      kolide_server_url: '',
      configured: false,
      sender_address: '',
      server: '',
      port: 587,
      authentication_type: 'authtype_username_password',
      user_name: '',
      password: '',
      enable_ssl_tls: true,
      authentication_method: 'authmethod_plain',
      verify_ssl_certs: true,
      enable_start_tls: true,
      email_enabled: false,
    };

    it('splits config into categories for the server', () => {
      expect(formatConfigDataForServer(config)).toEqual(omit(configStub, ['smtp_settings.configured']));
    });
  });

  describe('#formatSelectedTargetsForApi', () => {
    const { formatSelectedTargetsForApi } = helpers;

    it('splits targets into labels and hosts', () => {
      const targets = [host1, host2, label1, label2];

      expect(formatSelectedTargetsForApi(targets)).toEqual({
        hosts: [6, 5],
        labels: [1, 2],
      });
    });

    it('appends `_id` when appendID is specified', () => {
      const targets = [host1, host2, label1, label2];

      expect(formatSelectedTargetsForApi(targets, true)).toEqual({
        host_ids: [6, 5],
        label_ids: [1, 2],
      });
    });
  });

  describe('#setupData', () => {
    const formData = {
      email: 'hi@gnar.dog',
      name: 'Gnar Dog',
      kolide_server_url: 'https://gnar.kolide.co',
      org_logo_url: 'https://thegnar.co/assets/logo.png',
      org_name: 'The Gnar Co.',
      password: 'p@ssw0rd',
      password_confirmation: 'p@ssw0rd',
      username: 'gnardog',
    };

    it('formats the form data to send to the server', () => {
      expect(helpers.setupData(formData)).toEqual({
        kolide_server_url: 'https://gnar.kolide.co',
        org_info: {
          org_logo_url: 'https://thegnar.co/assets/logo.png',
          org_name: 'The Gnar Co.',
        },
        admin: {
          admin: true,
          email: 'hi@gnar.dog',
          name: 'Gnar Dog',
          password: 'p@ssw0rd',
          password_confirmation: 'p@ssw0rd',
          username: 'gnardog',
        },
      });
    });
  });
});
