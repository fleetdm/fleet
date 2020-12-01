import { mount } from 'enzyme';

import AppSettingsPage from 'pages/admin/AppSettingsPage';
import { flatConfigStub } from 'test/stubs';
import testHelpers from 'test/helpers';

const { connectedComponent, reduxMockStore } = testHelpers;
const baseStore = {
  app: { config: flatConfigStub, enrollSecret: [] },
};
const storeWithoutSMTPConfig = {
  ...baseStore,
  app: {
    config: { ...flatConfigStub, configured: false },
    enrollSecret: [],
  },
};
const storeWithSMTPConfig = {
  ...baseStore,
  app: {
    config: { ...flatConfigStub, configured: true },
    enrollSecret: [],
  },
};

describe('AppSettingsPage - component', () => {
  it('renders', () => {
    const mockStore = reduxMockStore(baseStore);
    const page = mount(connectedComponent(AppSettingsPage, { mockStore }));

    expect(page.find('AppSettingsPage').length).toEqual(1);
  });

  it('renders a warning if SMTP has not been configured', () => {
    const mockStore = reduxMockStore(storeWithoutSMTPConfig);
    const page = mount(
      connectedComponent(AppSettingsPage, { mockStore }),
    ).find('AppSettingsPage');

    const smtpWarning = page.find('WarningBanner');

    expect(smtpWarning.length).toEqual(1);
    expect(smtpWarning.find('Icon').length).toEqual(1);
    expect(smtpWarning.text()).toContain('Email is not currently configured in Fleet');
  });

  it('dismisses the smtp warning when "DISMISS" is clicked', () => {
    const mockStore = reduxMockStore(storeWithoutSMTPConfig);
    const page = mount(
      connectedComponent(AppSettingsPage, { mockStore }),
    );

    const smtpWarning = page.find('WarningBanner');
    const dismissButton = smtpWarning.find('Button').first();

    dismissButton.simulate('click');

    expect(page.find('WarningBanner').html()).toBeFalsy();
  });

  it('does not render a warning if SMTP has been configured', () => {
    const mockStore = reduxMockStore(storeWithSMTPConfig);
    const page = mount(
      connectedComponent(AppSettingsPage, { mockStore }),
    ).find('AppSettingsPage');

    expect(page.find('WarningBanner').html()).toBeFalsy();
  });
});
