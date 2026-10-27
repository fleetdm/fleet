import {
  canTriggerAPNSPing,
  isAndroidBYO,
  isBYODAccountDrivenUserEnrollment,
  isBYODManualEnrollment,
  isEnrolledInMdm,
  isPersonalEnrollment,
  MDM_ENROLLMENT_STATUS_UI_MAP,
  MdmEnrollmentStatus,
  wasBYODEnrolled,
} from "./mdm";

describe("personal enrollment predicates", () => {
  it.each<[MdmEnrollmentStatus | null, boolean, boolean, boolean]>([
    // status, isPersonalEnrollment, isBYODManualEnrollment, isBYODAccountDrivenUserEnrollment
    ["On (personal)", true, false, true],
    ["On (manual - personal)", true, true, false],
    ["On (manual)", false, false, false],
    ["On (automatic)", false, false, false],
    ["On (company-owned)", false, false, false],
    ["Off", false, false, false],
    ["Pending", false, false, false],
    [null, false, false, false],
  ])("%s", (status, personal, manualBYOD, accountDriven) => {
    expect(isPersonalEnrollment(status)).toBe(personal);
    expect(isBYODManualEnrollment(status)).toBe(manualBYOD);
    expect(isBYODAccountDrivenUserEnrollment(status)).toBe(accountDriven);
  });

  it("treats both personal statuses as enrolled", () => {
    expect(isEnrolledInMdm("On (personal)")).toBe(true);
    expect(isEnrolledInMdm("On (manual - personal)")).toBe(true);
    expect(isEnrolledInMdm("Off")).toBe(false);
  });

  it("identifies Android work profile by On (personal)", () => {
    expect(isAndroidBYO("On (personal)")).toBe(true);
    expect(isAndroidBYO("On (manual - personal)")).toBe(false);
  });

  it("identifies a BYOD host by either status or, after unenrolling, the flag", () => {
    expect(wasBYODEnrolled("On (personal)")).toBe(true);
    expect(wasBYODEnrolled("On (manual - personal)")).toBe(true);
    expect(wasBYODEnrolled("Off", true)).toBe(true);
    expect(wasBYODEnrolled("Off", false)).toBe(false);
    expect(wasBYODEnrolled("On (manual)")).toBe(false);
  });

  it("allows an APNs ping for both personal statuses", () => {
    const host = (enrollment_status: MdmEnrollmentStatus) => ({
      platform: "ios" as const,
      mdm: { connected_to_fleet: true, enrollment_status },
    });
    expect(canTriggerAPNSPing(host("On (personal)"))).toBe(true);
    expect(canTriggerAPNSPing(host("On (manual - personal)"))).toBe(true);
    expect(canTriggerAPNSPing(host("Off"))).toBe(false);
  });

  it("gives each personal status its own filter value", () => {
    expect(MDM_ENROLLMENT_STATUS_UI_MAP["On (personal)"].filterValue).toBe(
      "personal"
    );
    expect(
      MDM_ENROLLMENT_STATUS_UI_MAP["On (manual - personal)"].filterValue
    ).toBe("manual-personal");
  });
});
