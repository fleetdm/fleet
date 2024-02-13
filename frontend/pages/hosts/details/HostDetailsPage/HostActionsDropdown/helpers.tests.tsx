import { canUnlock, IHostActionConfigOptions } from "./helpers";

interface ITestCase<T> {
  name: string;
  args: Partial<T>;
  expected: boolean;
}

describe("canUnlock function", () => {
  const testCases: ITestCase<IHostActionConfigOptions>[] = [
    {
      name: "Darwin, Unlocking, MDM Enabled/Configured/Enrolled",
      args: {
        hostPlatform: "darwin",
        hostMdmDeviceStatus: "unlocking",
        isMdmEnabledAndConfigured: true,
        isEnrolledInMdm: true,
      },
      expected: true,
    },
    {
      name: "Darwin, Locked, MDM Enabled/Configured/Enrolled",
      args: {
        hostPlatform: "darwin",
        hostMdmDeviceStatus: "locked",
        isMdmEnabledAndConfigured: true,
        isEnrolledInMdm: true,
      },
      expected: true,
    },
    {
      name: "Darwin, Unlocking, MDM Not Fully Configured",
      args: {
        hostPlatform: "darwin",
        hostMdmDeviceStatus: "unlocking",
        isMdmEnabledAndConfigured: false,
        isEnrolledInMdm: true,
      },
      expected: false,
    },
    {
      name: "Windows, Unlocking",
      args: {
        hostPlatform: "windows",
        hostMdmDeviceStatus: "unlocking",
      },
      expected: false,
    },
    {
      name: "Windows, Locked",
      args: {
        hostPlatform: "windows",
        hostMdmDeviceStatus: "locked",
      },
      expected: true,
    },
    {
      name: "Linux-like, Unlocking",
      args: {
        hostPlatform: "ubuntu",
        hostMdmDeviceStatus: "unlocking",
      },
      expected: false,
    },
    {
      name: "Linux-like, Locked",
      args: {
        hostPlatform: "ubuntu",
        hostMdmDeviceStatus: "locked",
      },
      expected: true,
    },
  ];

  testCases.forEach(({ name, args, expected }) => {
    it(name, () => {
      const fullArgs = Object.assign(
        {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isGlobalMaintainer: false,
          isGlobalObserver: false,
          isTeamAdmin: false,
          isTeamMaintainer: false,
          isTeamObserver: false,
          isFleetMdm: true,
          isEnrolledInMdm: true,
          isMdmEnabledAndConfigured: true,
          hostPlatform: "",
          hostMdmDeviceStatus: "",
        },
        args
      ) as IHostActionConfigOptions;
      expect(canUnlock(fullArgs)).toEqual(expected);
    });
  });
});
