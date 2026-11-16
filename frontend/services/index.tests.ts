import { http, HttpResponse } from "msw";

import mdmAPI from "services/entities/mdm";
import softwareAPI from "services/entities/software";
import mockServer from "test/mock-server";
import { baseUrl } from "test/test-utils";

const STORAGE_URL = "https://storage.example/uploads/up1";

// Records the presign body, the storage PUT's auth header, and the
// registration form so each test can assert which path ran.
const setup = (
  registerPath: string,
  method: "post" | "patch" = "post",
  { presignStatus = 200, storage = "ok" as "ok" | "403" | "network" } = {}
) => {
  const seen: {
    presign?: unknown;
    storageAuth?: string | null;
    form?: FormData;
  } = {};
  mockServer.use(
    http.post(baseUrl("/staged_upload"), async ({ request }) => {
      seen.presign = await request.json();
      return presignStatus === 200
        ? HttpResponse.json({ upload_id: "up1", url: STORAGE_URL })
        : HttpResponse.json(
            {
              errors: [
                { name: "base", reason: "Direct upload isn't available." },
              ],
            },
            { status: presignStatus }
          );
    }),
    http.put(STORAGE_URL, ({ request }) => {
      seen.storageAuth = request.headers.get("Authorization");
      if (storage === "network") return HttpResponse.error();
      return new HttpResponse(null, { status: storage === "ok" ? 200 : 403 });
    }),
    http[method](baseUrl(registerPath), async ({ request }) => {
      seen.form = await request.formData();
      return HttpResponse.json({ software_package: { title_id: 1 } });
    })
  );
  return seen;
};

const file = new File(["abc"], "a.pkg");

describe("direct upload to storage", () => {
  it("add package uploads to storage, then registers the upload id", async () => {
    const seen = setup("/software/package");
    await softwareAPI.addSoftwarePackage({
      data: { software: file, selfService: false } as never,
      teamId: 3,
      directUpload: true,
    });
    expect(seen.presign).toEqual({
      target: "software_package",
      fleet_id: 3,
      size: 3,
    });
    expect(seen.storageAuth).toBeNull();
    expect(seen.form?.get("upload_id")).toBe("up1");
    expect(seen.form?.get("filename")).toBe("a.pkg");
    expect(seen.form?.has("software")).toBe(false);
  });

  it("add package sends the file to Fleet when direct upload is off", async () => {
    const seen = setup("/software/package");
    await softwareAPI.addSoftwarePackage({
      data: { software: file, selfService: false } as never,
      teamId: 3,
    });
    expect(seen.presign).toBeUndefined();
    expect(seen.form?.has("software")).toBe(true);
    expect(seen.form?.has("upload_id")).toBe(false);
  });

  it.each(["403", "network"] as const)(
    "a %s storage failure rejects without registering or sending the file to Fleet",
    async (storage) => {
      const seen = setup("/software/package", "post", { storage });
      await expect(
        softwareAPI.addSoftwarePackage({
          data: { software: file, selfService: false } as never,
          teamId: 3,
          directUpload: true,
        })
      ).rejects.toBeDefined();
      expect(seen.storageAuth).toBeNull();
      expect(seen.form).toBeUndefined();
    }
  );

  it("a failed upload URL request stops before the storage PUT", async () => {
    const seen = setup("/bootstrap", "post", { presignStatus: 400 });
    await expect(
      mdmAPI.uploadBootstrapPackage(file, 0, true)
    ).rejects.toMatchObject({ status: 400 });
    expect(seen.storageAuth).toBeUndefined();
    expect(seen.form).toBeUndefined();
  });

  it("edit package uploads a replacement file to storage", async () => {
    const seen = setup("/software/titles/5/package", "patch");
    await softwareAPI.editSoftwarePackage({
      data: {
        software: file,
        selfService: false,
        targetType: "All hosts",
      } as never,
      orignalPackage: {} as never,
      softwareId: 5,
      teamId: 3,
      directUpload: true,
    });
    expect(seen.storageAuth).toBeNull();
    expect(seen.form?.get("upload_id")).toBe("up1");
    expect(seen.form?.get("filename")).toBe("a.pkg");
    expect(seen.form?.has("software")).toBe(false);
  });

  it("bootstrap package uploads to storage, then registers the upload id", async () => {
    const seen = setup("/bootstrap");
    await mdmAPI.uploadBootstrapPackage(file, 0, true);
    expect(seen.presign).toEqual({
      target: "bootstrap_package",
      fleet_id: 0,
      size: 3,
    });
    expect(seen.storageAuth).toBeNull();
    expect(seen.form?.get("upload_id")).toBe("up1");
    expect(seen.form?.get("filename")).toBe("a.pkg");
    expect(seen.form?.has("package")).toBe(false);
  });

  it("bootstrap package sends the file to Fleet when direct upload is off", async () => {
    const seen = setup("/bootstrap");
    await mdmAPI.uploadBootstrapPackage(file, 0);
    expect(seen.presign).toBeUndefined();
    expect(seen.form?.has("package")).toBe(true);
  });
});
