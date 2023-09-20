import Table from "./Table";

export default class TableUsers extends Table {
  name = "users";
  columns = ["uid", "username", "email"];

  async generate() {
    const { email, id } = await chrome.identity.getProfileUserInfo({});
    return { data: [{ uid: id, email, username: email }] };
  }
}
