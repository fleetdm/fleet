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

describe("convertToCSV - utility", () => {
  it("converts an array of objects to CSV format", () => {
    expect(convertToCSV(objArray)).toEqual(
      '"first_name","last_name"\n"Mike","Stone"\n"Paul","Simon"'
    );
  });
});
