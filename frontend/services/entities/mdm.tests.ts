import sendRequest from "services";

import mdmAPI from "./mdm";

jest.mock("services");

const mockedSendRequest = sendRequest as jest.MockedFunction<
  typeof sendRequest
>;

const profileFile = () =>
  new File(["{}"], "declaration.json", { type: "application/json" });

/** the FormData the last sendRequest call was given. */
const sentFormData = () => mockedSendRequest.mock.calls[0][2] as FormData;

describe("mdmAPI.uploadProfile", () => {
  beforeEach(() => {
    mockedSendRequest.mockResolvedValue({});
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it("omits the activation when none was written", async () => {
    // the editor shows the example as a placeholder, so an untouched field is an
    // empty string. Sending it would be rejected by the API, which requires the
    // activation to reference the uploaded declaration.
    await mdmAPI.uploadProfile({
      file: profileFile(),
      teamId: 0,
      customActivation: "",
    });

    expect(sentFormData().has("activation")).toBe(false);
  });

  it("omits the activation when it is only whitespace", async () => {
    await mdmAPI.uploadProfile({
      file: profileFile(),
      teamId: 0,
      customActivation: "   \n  ",
    });

    expect(sentFormData().has("activation")).toBe(false);
  });

  it("omits the activation when the caller doesn't pass one", async () => {
    await mdmAPI.uploadProfile({ file: profileFile(), teamId: 0 });

    expect(sentFormData().has("activation")).toBe(false);
  });

  it("sends a written activation as a named file part", async () => {
    // the API reads this from the multipart File map rather than Value, so it
    // has to arrive as a Blob with a filename -- a plain string field would be
    // silently ignored.
    const activation = '{"Type":"com.apple.activation.simple"}';

    await mdmAPI.uploadProfile({
      file: profileFile(),
      teamId: 0,
      customActivation: activation,
    });

    // NOTE: not an instanceof File assertion -- jsdom and the test realm have
    // separate File constructors, so it fails with "Expected: File, Received:
    // File". Asserting the filename and contents covers what matters: a plain
    // string field would have neither.
    const sent = sentFormData().get("activation") as File;
    expect(typeof sent).not.toEqual("string");
    expect(sent.name).toEqual("activation.json");
    expect(await sent.text()).toEqual(activation);
  });
});
