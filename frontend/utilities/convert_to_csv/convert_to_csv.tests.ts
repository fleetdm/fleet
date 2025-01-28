import convertToCSV from "utilities/convert_to_csv";

const objArray = [
  {
    first_name: "Mike",
    last_name: "Stone",
  },
  {
    first_name: "Paul",
    last_name: "Simon",
  },
];

// tests json known value edge case and hypothetical key edge case
const objArray2 = [
  {
    host_display_name: "Rachel@Fleet",
    last_fetched: "2024-06-25T13:11:18Z",
    uid: "145",
    json_result: {
      "AC Power:": {
        acwake: "0",
        hibernatefile: "/var/vm/sleepimage",
      },
    },
    'edge","case': "true",
  },
];

const tableHeaders = [
  { id: "host_display_name", sortType: "caseInsensitive" },
  { id: "last_fetched", sortType: "caseInsensitive" },
  { id: "uid", sortType: "alphanumeric" },
  { id: "json_result", sortType: "caseInsensitive" },
  { id: 'edge","case', sortType: "caseInsensitve" },
];

describe("convertToCSV - utility", () => {
  it("converts an array of objects to CSV format", () => {
    expect(convertToCSV({ objArray })).toEqual(
      '"first_name","last_name"\n"Mike","Stone"\n"Paul","Simon"'
    );
  });
  it("correctly creates table headers and fields including quotes and commas to CSV format", () => {
    expect(convertToCSV({ objArray: objArray2, tableHeaders })).toEqual(
      '"host_display_name","last_fetched","uid","json_result","edge"",""case"\n"Rachel@Fleet","2024-06-25T13:11:18Z","145","{""AC Power:"":{""acwake"":""0"",""hibernatefile"":""/var/vm/sleepimage""}}","true"'
    );
  });
});
