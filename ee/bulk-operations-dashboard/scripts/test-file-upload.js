module.exports = {


  friendlyName: 'Test file transfers',


  description: '',


  fn: async function () {
    var WritableStream = require('stream').Writable;
    let { Readable } = require('stream');
    let axios = require('axios');

    sails.log('copying file Fleet instance » Fleet instance (other team)');
    let softwareApiUrl = `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/7/package?alt=media&team_id=2`;
    await sails.cp(softwareApiUrl, {
      adapter: () => {
        return {
          ls: undefined,
          rm: undefined,
          receive: undefined,
          read: (softwareApiUrl) => {
            // Create a readable stream
            const readable = new Readable({
              read() {
                // Empty _read method; we'll handle data pushing with events below
              }
            });

            // Now we'll fetch the data asynchronously and pipe it into the readable stream
            (async () => {
              try {
                const streamResponse = await axios({
                  url: softwareApiUrl,
                  method: 'GET',
                  responseType: 'stream',
                  headers: {
                    Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
                  },
                });

                console.log('Received stream from API, piping data...');

                // Pipe data from the response stream into the readable stream
                streamResponse.data.on('data', (chunk) => {
                  const canContinue = readable.push(chunk);
                  if (!canContinue) {
                    streamResponse.data.pause();  // Pause if we can't push more data
                  }
                });

                // Resume the stream when readable is ready
                readable.on('drain', () => {
                  streamResponse.data.resume();
                });

                // When the source stream ends, we signal end of the readable stream
                streamResponse.data.on('end', () => {
                  readable.push(null); // Signal end of stream
                });

                // Handle any errors from the source stream
                streamResponse.data.on('error', (err) => {
                  readable.emit('error', err); // Propagate the error to the readable stream
                });

              } catch (error) {
                console.error('Error during read operation:', error);
                readable.emit('error', new Error('Failed to download file: ' + error.message));
              }
            })();

            return readable;
          },
        };
      },
    },
    {
      adapter: ()=>{
        return {
          ls: undefined,
          rm: undefined,
          read: undefined,
          receive: (unusedOpts)=>{
            // This `_write` method is invoked each time a new file is received
            // from the Readable stream (Upstream) which is pumping filestreams
            // into this receiver.  (filename === `__newFile.filename`).
            var receiver__ = WritableStream({ objectMode: true });
            // Create a new drain (writable stream) to send through the individual bytes of this file.
            receiver__._write = (__newFile, encoding, doneWithThisFile)=>{
              let axios = require('axios');
              let FormData = require('form-data');
              let form = new FormData();
              form.append('team_id', 0);
              form.append('software', __newFile, {
                filename: 'test.exe',
                contentType: 'application/octet-stream'
              });
              (async ()=>{
                try {
                  await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
                    headers: {
                      Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
                      ...form.getHeaders()
                    },
                  });
                } catch(error){
                  throw new Error('Failed to upload file:'+ require('util').inspect(error, {depth: null}));
                }
              })()
              .then(()=>{
                console.log('ok supposedly a file is finished uploading');
                doneWithThisFile();
              })
              .catch((err)=>{
                doneWithThisFile(err);
              });
            };//ƒ
            return receiver__;
          }
        };
      },
    });

    let software = await UndeployedSoftware.find();
    let uploadedSoftware = software[0];
    sails.log(`Uploading file S3 » Fleet instance`);
    await sails.cp(uploadedSoftware.fd, {}, {
      adapter: ()=>{
        return {
          ls: undefined,
          rm: undefined,
          read: undefined,
          receive: (unusedOpts)=>{
            // This `_write` method is invoked each time a new file is received
            // from the Readable stream (Upstream) which is pumping filestreams
            // into this receiver.  (filename === `__newFile.filename`).
            var receiver__ = WritableStream({ objectMode: true });
            // Create a new drain (writable stream) to send through the individual bytes of this file.
            receiver__._write = (__newFile, encoding, doneWithThisFile)=>{
              let axios = require('axios');
              let FormData = require('form-data');
              let form = new FormData();
              form.append('team_id', 0);
              form.append('software', __newFile, {
                filename: uploadedSoftware.filename,
                contentType: 'application/octet-stream'
              });
              (async ()=>{
                try {
                  await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
                    headers: {
                      Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
                      ...form.getHeaders()
                    },
                  });
                } catch(error){
                  throw new Error('Failed to upload file:'+ require('util').inspect(error, {depth: null}));
                }
              })()
              .then(()=>{
                console.log('ok supposedly a file is finished uploading');
                doneWithThisFile();
              })
              .catch((err)=>{
                doneWithThisFile(err);
              });
            };//ƒ
            return receiver__;
          }
        };
      }
    });




    // Uses both things
    // await sails.cp(uploadedFile.fd, {
    //   adapter: ()=>{
    //     return {
    //       ls: undefined,
    //       rm: undefined,
    //       receive: undefined,
    //       read: (dowloadApiUrl) => {
    //         let stream = new require('stream').PassThrough(); // Create a PassThrough stream (readable)

    //         (async () => {
    //           try {
    //             let response = await sails.helpers.http.getStream.with({
    //               url: dowloadApiUrl,
    //               headers: {
    //                 Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
    //               },
    //             });

    //             if (response && typeof response.pipe === 'function') {
    //               console.log('piping this response')
    //               response.pipe(stream);  // Pipe the response stream to the PassThrough stream
    //             } else {
    //               // stream.emit('error', new Error('No valid stream returned from the API.'));
    //             }
    //           } catch (error) {
    //             throw new Error(error);
    //             // stream.emit('error', new Error('Failed to download file: ' + require('util').inspect(error, { depth: null })));
    //           }
    //         })();

    //         return stream;  // Return the PassThrough readable stream immediately to `sails.cp()`
    //       }
    //     }
    //   },
    // },
    // {
    //   adapter: ()=>{
    //     return {
    //       ls: undefined,
    //       rm: undefined,
    //       read: undefined,
    //       receive: (unusedOpts)=>{
    //         // This `_write` method is invoked each time a new file is received
    //         // from the Readable stream (Upstream) which is pumping filestreams
    //         // into this receiver.  (filename === `__newFile.filename`).
    //         var receiver__ = new WritableStream({ objectMode: true });
    //         // Create a new drain (writable stream) to send through the individual bytes of this file.
    //         receiver__._write = (__newFile, encoding, doneWithThisFile)=>{
    //           let axios = require('axios');
    //           let FormData = require('form-data');
    //           let form = new FormData();
    //           form.append('team_id', team);
    //           form.append('software', __newFile, {
    //             filename: software.name,
    //             contentType: 'application/octet-stream'
    //           });
    //           (async ()=>{
    //             try {
    //               await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
    //                 headers: {
    //                   Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
    //                   ...form.getHeaders()
    //                 },
    //               })
    //             } catch(error){
    //               throw new Error('Failed to upload file:'+ require('util').inspect(error, {depth: null}));
    //             }
    //           })()
    //           .then(()=>{
    //             console.log('ok supposedly a file is finished uploading');
    //             doneWithThisFile();
    //           })
    //           .catch((err)=>{
    //             doneWithThisFile(err);
    //           });
    //         };//ƒ
    //         return receiver__;
    //       }
    //     }
    //   },
    // });


    // await sails.cp(softwareApiUrl, {
    //       adapter: ()=>{
    //         return {
    //           ls: undefined,
    //           rm: undefined,
    //           receive: undefined,
    //           read: (softwareApiUrl) => {
    //             (async ()=>{
    //               try {
    //                 return await sails.helpers.http.getStream.with({
    //                   url: softwareApiUrl,
    //                   headers: {
    //                     Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
    //                   },
    //                 });
    //               } catch(error){
    //                 throw new Error('Failed to download file:'+ require('util').inspect(error, {depth: null}));
    //               }
    //             })()
    //             .then(()=>{
    //               console.log('ok supposedly a file is finished downloading');
    //             })
    //             .catch((err)=>{
    //               throw new Error(err)
    //             });
    //           },
    //         }
    //       },
    //     },
    //     {
    //       adapter: ()=>{
    //         return {
    //           ls: undefined,
    //           rm: undefined,
    //           read: undefined,
    //           receive: (unusedOpts)=>{
    //             // This `_write` method is invoked each time a new file is received
    //             // from the Readable stream (Upstream) which is pumping filestreams
    //             // into this receiver.  (filename === `__newFile.filename`).
    //             var receiver__ = WritableStream({ objectMode: true });
    //             // Create a new drain (writable stream) to send through the individual bytes of this file.
    //             receiver__._write = (__newFile, encoding, doneWithThisFile)=>{
    //               let axios = require('axios');
    //               let FormData = require('form-data');
    //               let form = new FormData();
    //               form.append('team_id', 0);
    //               form.append('software', __newFile, {
    //                 filename: 'test.exe',
    //                 contentType: 'application/octet-stream'
    //               });
    //               (async ()=>{
    //                 try {
    //                   await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
    //                     headers: {
    //                       Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
    //                       ...form.getHeaders()
    //                     },
    //                   })
    //                 } catch(error){
    //                   throw new Error('Failed to upload file:'+ require('util').inspect(error, {depth: null}));
    //                 }
    //               })()
    //               .then(()=>{
    //                 console.log('ok supposedly a file is finished uploading');
    //                 doneWithThisFile();
    //               })
    //               .catch((err)=>{
    //                 doneWithThisFile(err);
    //               });
    //             };//ƒ
    //             return receiver__;
    //           }
    //         }
    //       },
    //     });





    // // from Fleet instance to s3:
    // let fi = await sails.cp(softwareApiUrl, {
    //   adapter: ()=>{
    //     return {
    //       ls: undefined,
    //       rm: undefined,
    //       read: (softwareApiUrl) => {
    //         // Create a Readable stream
    //         let readableStream = new Readable({
    //           read(size) {
    //             // Make an async call to get data from the software API
    //             (async () => {
    //               try {
    //                 let response = await axios.get(softwareApiUrl, {
    //                   responseType: 'stream', // Ensure the response is a stream
    //                   headers: {
    //                     Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
    //                   }
    //                 });

    //                 // Pipe the response data into this stream
    //                 response.data.on('data', (chunk) => {
    //                   this.push(chunk); // Push each chunk into the readable stream
    //                 });

    //                 response.data.on('end', () => {
    //                   this.push(null); // Signal end of stream
    //                 });

    //               } catch (error) {
    //               }
    //             })();
    //           }
    //         });

    //         return readableStream; // Return the readable stream
    //       },
    //     }
    //   },
    // })
    // console.log(fi);


    // let downloading = await sails.helpers.http.getStream.with({
    //   url: `${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/titles/4/package?alt=media&team_id=3`,
    //   headers: {
    //     Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
    //   },
    // });
    // await sails.upload(downloading, {
    //   adapter: ()=>{
    //     return {
    //       ls: undefined,
    //       rm: undefined,
    //       read: undefined,
    //       receive: (unusedOpts)=>{
    //         // This `_write` method is invoked each time a new file is received
    //         // from the Readable stream (Upstream) which is pumping filestreams
    //         // into this receiver.  (filename === `__newFile.filename`).
    //         var receiver__ = WritableStream({ objectMode: true });
    //         // Create a new drain (writable stream) to send through the individual bytes of this file.
    //         receiver__._write = (__newFile, encoding, doneWithThisFile)=>{
    //           // var newFileDrain__ = fsx.createWriteStream(`${sails.config.appPath}/assets/foobar.fake`, encoding);
    //           let axios = require('axios');
    //           let FormData = require('form-data');
    //           let form = new FormData();
    //           form.append('team_id', 1);
    //           form.append('software', __newFile, {
    //             filename: 'foo.exe',
    //             contentType: 'application/octet-stream'
    //           });

    //           (async ()=>{
    //             try {

    //               // await sails.helpers.http.sendHttpRequest.with({
    //               //   method: 'POST',
    //               //   baseUrl: sails.config.custom.fleetBaseUrl,
    //               //   url: `/api/v1/fleet/software/package?team_id=2`,
    //               //   enctype: 'multipart/form-data',
    //               //   body: form,
    //               //   headers: {
    //               //     Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
    //               //     ...form.getHeaders()
    //               //   },
    //               // });
    //               await axios.post(`${sails.config.custom.fleetBaseUrl}/api/v1/fleet/software/package`, form, {
    //                 headers: {
    //                   Authorization: `Bearer ${sails.config.custom.fleetApiToken}`,
    //                   ...form.getHeaders()
    //                 },
    //               })
    //             } catch(error){
    //               throw new Error('Failed to upload file:'+ require('util').inspect(error, {depth: null}));
    //             }
    //           })()
    //           .then(()=>{
    //             console.log('ok supposedly a file is finished uploading');
    //             doneWithThisFile();
    //             // newFileDrain__.on('finish', ()=>{
    //             //   receiver__.emit('writefile', __newFile);// Indicate that a file was persisted.
    //             //   console.log('ok supposedly a file is finished uploading');
    //             //   doneWithThisFile();
    //             // });
    //             // __newFile.pipe(newFileDrain__);
    //           })
    //           .catch((err)=>{
    //             doneWithThisFile(err);
    //           });

    //         };//ƒ

    //         return receiver__;
    //       }
    //     }
    //   }
    // });

    // console.log('ok supposedly everything is now uploaded');
  }
};

