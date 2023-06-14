// import Table from "./Table";

// enum IdleState {
//   "active",
//   "idle",
//   "locked",
// }

// export default class TableSystemState extends Table {
//   name = "system_state";
//   columns = ["idle_state"];

//   async generate() {
//     let idle_state;

//     // This uses an old function callback so need to convert the callback style to a promise
//     // Consider updating the version so we don't have to use a function callback
//     try {
//       const delay: number = await new Promise((resolve) =>
//         chrome.idle.getAutoLockDelay(resolve)
//       );

//       const idle_state = await new Promise((resolve) =>
//         chrome.idle.queryState(delay, resolve)
//       );

//       return [{ idle_state }];
//     } catch (err) {
//       console.warn("get system state info:", err);
//     }

//     return [{ idle_state }];
//   }
// }
