import { AxiosError } from "axios";

const DEFAULT_AXIOS_ERROR_MOCK: AxiosError = {
  isAxiosError: true,
  toJSON: () => ({}),
  name: "Error",
  message: "error message",
};

const createMockAxiosError = (overrides?: Partial<AxiosError>): AxiosError => {
  return { ...DEFAULT_AXIOS_ERROR_MOCK, ...overrides };
};

export default createMockAxiosError;
