import valid_hostname from "./valid_hostname";

describe("valid_hostname", () => {
  it("correctly validates supported hostnames and IP formats", () => {
    type TestCase = [string, boolean];
    const testCases: TestCase[] = [
      ["fleet.example.com", true], // Standard FQDN
      ["fleet.example.com:8090", true], // FQDN with port
      ["192.168.0.1", true], // IPv4
      ["192.168.0.1:9090", true], // IPv4 with port
      ["2001:0db8:85a3:0000:0000:8a2e:0370:7334", true], // IPv6 (no brackets)
      ["[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080", true], // IPv6 with brackets and port
      ["localhost", true], // Localhost
      ["localhost:3000", true], // Localhost with port
      ["not a valid url!", false], // Gibberish
      ["example.com:", false], // Missing port after colon
      ["example.com:70000", false], // Invalid port number (> 65535)
      ["256.256.256.256", false], // Invalid IPv4
      ["2001:xyz:123", false], // Invalid IPv6
      [":8080", false], // Missing host
      ["[2001:db8::1]", false], // IPv6 with brackets but no port (typically invalid for host input)
    ];
    testCases.forEach(([value, expected]) => {
      expect(valid_hostname(value)).toBe(expected);
    });
  });
});
