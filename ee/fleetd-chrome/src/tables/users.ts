import Table from "./Table";

export default class TableUsers extends Table {
  name = "users";
  // TODO remove type, gruopname, shell, gid, directory after updating users detail query in Fleet.
  columns = [
    "uid",
    "username",
    "type",
    "groupname",
    "shell",
    "email",
    "gid",
    "directory",
  ];

  async generate() {
    const { email, id } = await chrome.identity.getProfileUserInfo({});
    return [
      {
        uid: id,
        email,
        username: email,
        type: "",
        groupname: "",
        shell: "",
        gid: "",
        directory: "",
      },
    ];
  }
}
