const fs = require('fs');
const path = require('path');

const args = process.argv.slice(2);
const directoryArg = args[0];

const directories = fs.readdirSync(directoryArg);

directories.forEach(directory => {

  const filepath = path.resolve(__dirname, directoryArg, directory, 'index.js');
  // console.log(path.resolve(__dirname, directoryArg, directory, 'index.js'));

  fs.readFile(filepath, 'utf8', (readErr, data) => {
    if (readErr) {
      return console.log(readErr);
    }
    const result = data.replace(/default/g, '{ default }');

    fs.writeFile(filepath, result, 'utf8', (writeErr) => {
      if (writeErr) return console.log(writeErr);
    });
  });
});


// fs.readFile(file, 'utf8', (readErr, data) => {
//   if (readErr) {
//     return console.log(readErr);
//   }
//   const result = data.replace(/default/g, '{ default }');
//
//   fs.writeFile(file, result, 'utf8', (writeErr) => {
//     if (writeErr) return console.log(writeErr);
//   });
// });
