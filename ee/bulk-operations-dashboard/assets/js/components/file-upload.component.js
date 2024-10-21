/**
 * <file-upload>
 * -----------------------------------------------------------------------------
 * A form component which includes a file input,
 * and handles showing a preview image when a file is selected.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 * @slot image-upload-instructions
 *                  Optional override for the HTML to display next to the image.
 *                  (Only relevant when mode="image".)
 * -----------------------------------------------------------------------------
 * @event input   - emitted when a file upload is selectd or cleared privately
 *                  (i.e. using the native file picker).  May be used implicitly
 *                  with v-model -- e.g.:
 *                      v-model="formData.logoUpload"
 *
 *                  And also explicitly with an @input listener -- e.g.:
 *                      @input="inputLogoUpload($event)"
 *
 *                  In either case, the handler is passed a File instance.
 *                  File instances can be passed around directly to a lot of
 *                  things (like parasails/Cloud SDK), and you can also extract
 *                  a data URI string from them using either FileReader's
 *                  `.readAsDataURL()` or URL's `createObjectURL(file)`.
 *                  Links:
 *                   • https://developer.mozilla.org/en-US/docs/Web/API/FileReader/readAsDataURL
 *                   • https://developer.mozilla.org/en-US/docs/Web/API/File/Using_files_from_web_applications#Example_Using_object_URLs_to_display_images#Example_Using_object_URLs_to_display_images
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('fileUpload', {

  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'mode', //« Tells us us whether this component should be mounted in the default "file" mode (miscellaneous file) or mounted with an image previewer in "image" mode.
    'disabled',//« for disabling from the outside (e.g. while syncing)

    'value',//« for v-model -- should not be used to set initial value

    // Note that, for now, `value` is completely separate from initialFileName,
    // initialMimeType, initialFileSize, and initialSrc.  These props are how
    // you indicate the initial value for this file upload field.
    // > FUTURE: Find some way to unify these props with `value` aka v-model
    // > (see also other FUTURE note below about "value" watcher)
    'initialFileName',// « file name (basename including extension; no path)
    'initialFileMimeType',// « the file's MIME type (string)
    'initialFileSize',// « number of bytes (positive integer)
    'placeholderImageSrc',// « the placeholder image to display when no image is selected (e.g. a custom silhouette)
    'initialSrc',// «Conventional approach is to either (A) prepare this on the
    // backend so it can use the configured baseUrl and/or cache-busting, then
    // pass that in here, or (B) to use a root-relative URL.  Either way, we
    // always provide the proper dynamic URL here if we have one and just let
    // the corresponding download action take care of either grabbing the real
    // dynamic file or streaming a static placeholder file from disk (e.g. a
    // fake avatar).  **THAT SAID:** If this initialSrc is omitted, then a
    // baked-in placeholder image is used instead.  That allows the UI to
    // display a file-upload-previewer-specific icon (by default, a photo icon)
    'buttonClass',//« any classes to include on the button other than 'file-upload-button'.
    // defaults to 'btn btn-outline-primary'
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      isEmpty: false,// « whether or not this upload field is empty
      previewSrc: undefined,// « determined by initialSrc or the bytes from a selected file upload
      isCurrentlyDisabled: false, //« controlled by watching `disabled` prop
      isReadingFileUpload: false, //« spinlock
      selectedFileName: undefined,
      selectedFileMimeType: undefined,
      selectedFileIconClass: undefined,
      selectedFileSize: undefined,
      uploadedFilename: undefined,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="clearfix" :class="[mode === 'image' ? 'image-mode' : 'file-mode', isCurrentlyDisabled ? 'disabled' : '']">
    <div v-if="mode === 'profiles'">
      <div purpose="profile-upload-input" v-if="isEmpty">
        <div class="d-flex flex-column align-items-center">
          <!-- <input id='file-upload' type="file" > -->
          <img style="height: 40px; width: 34px;" src="/images/profile-34x40@2x.png">
          <p><strong>Upload configuration profile</strong></p>
          <p class="muted">.mobileconfig and .json for macOS, iOS, and iPadOS.</p>
          <p class="muted">.xml for Windows.</p>
          <div class="btn-and-tips-if-relevant">
            <label purpose="file-upload" for="file-upload-input">
              <img src="/images/upload-16x17@2x.png" style="height: 16px; width: 16px; margin-right: 8px">Choose File
            </label>
            <input id="file-upload-input" type="file" class="file-input d-none" :disabled="isCurrentlyDisabled" accept=".xml,.mobileconfig,.json" @change="changeFileInput($event)"/>
          </div>
        </div>
      </div>
      <div purpose="profile-information" v-else>
        <div class="d-flex flex-row justify-content-start">
          <img style="height: 40px; width: 34px;" src="/images/profile-34x40@2x.png">
          <div class="d-flex flex-column">
            <p><strong>{{selectedFileName.replace(/\.(xml|mobileconfig|json)$/g, '').replace(/^\d{4}-\d{2}-\d{2}_/, '')}}</strong></p>
            <p class="muted" v-if="_.endsWith(selectedFileName, 'xml')">Windows</p>
            <p class="muted" v-else>macOS, iOS, iPadOS</p>
          </div>
        </div>
      </div>
    </div>
    <div v-else-if="mode === 'scripts'">
      <div purpose="script-upload-input" v-if="isEmpty">
        <div class="d-flex flex-column align-items-center">
          <!-- <input id='file-upload' type="file" > -->
          <div class="d-flex flex-row justify-content-center mb-2">
            <img style="height: 40px; width: 34px; margin-right: 16px;" src="/images/script-icon-sh-34x40@2x.png">
            <img style="height: 40px; width: 34px;" src="/images/script-icon-ps1-34x40@2x.png">
          </div>
          <p class="mb-2">Shell (.sh) for macOS and Linux or PowerShell (.ps1) for Windows</p>
          <p style="color: #8B8FA2" class="muted">By default, scripts will run with “#!/bin/sh” on macOS and Linux. </p>
          <div class="btn-and-tips-if-relevant">
            <label purpose="file-upload" for="file-upload-input">
              <img src="/images/upload-16x17@2x.png" style="height: 16px; width: 16px; margin-right: 8px">Choose file
            </label>
            <input id="file-upload-input" type="file" class="file-input d-none" :disabled="isCurrentlyDisabled" accept=".sh,.ps1" @change="changeFileInput($event)"/>
          </div>
        </div>
      </div>
      <div purpose="script-information" v-else>
        <div class="d-flex flex-row justify-content-start">
          <img style="height: 40px; width: 34px;" src="/images/script-icon-ps1-34x40@2x.png" v-if="_.endsWith(selectedFileName, 'ps1')">
          <img style="height: 40px; width: 34px;" src="/images/script-icon-sh-34x40@2x.png" v-else>
          <div class="d-flex flex-column">
            <p><strong>{{selectedFileName}}</strong></p>
            <p class="muted" v-if="_.endsWith(selectedFileName, 'ps1')">Windows</p>
            <p class="muted" v-else>macOS & Linux</p>
          </div>
        </div>
      </div>
    </div>
    <div v-else-if="mode === 'software'">
      <div purpose="software-upload-input" v-if="isEmpty">
        <div class="d-flex flex-column align-items-center">
          <div class="d-flex flex-row justify-content-center mb-2">
            <img style="height: 40px; width: 34px; margin-right: 16px;" src="/images/software-icon-34x40@2x.png">
          </div>
          <p style="color: #8B8FA2" class="muted">.pkg, .msi, .exe, or .deb</p>
          <div class="btn-and-tips-if-relevant d-flex flex-row justify-content-center mt-0">
            <label purpose="file-upload" for="file-upload-input">
              <img src="/images/upload-16x17@2x.png" style="height: 16px; width: 16px; margin-right: 8px">Choose file
            </label>
            <input id="file-upload-input" type="file" class="file-input d-none" :disabled="isCurrentlyDisabled" accept=".exe,.pkg,.deb,.msi" @change="changeFileInput($event)"/>
          </div>
        </div>
      </div>
      <div purpose="software-information" v-else>
        <div class="d-flex flex-row justify-content-start">
          <img style="height: 40px; width: 34px; margin-right: 16px;" src="/images/software-icon-34x40@2x.png">
          <div class="d-flex flex-column">
            <p><strong>{{selectedFileName}}</strong></p>
            <p class="muted" v-if="_.endsWith(selectedFileName, '.exe') || _.endsWith(selectedFileName, '.msi')">Windows</p>
            <p class="muted" v-else-if="_.endsWith(selectedFileName, '.pkg')">macOS</p>
            <p class="muted" v-else-if="_.endsWith(selectedFileName, '.deb')">Linux</p>
          </div>
        </div>
      </div>
    </div>
    <div v-else-if="mode === 'software-pencil'">
      <div class="btn-and-tips-if-relevant">
        <label purpose="file-upload" for="file-upload-input">
          <img src="/images/icon-edit-software-16x16@2x.png" style="height: 16px; margin-right: 8px">
        </label>
        <input id="file-upload-input" type="file" class="file-input d-none" :disabled="isCurrentlyDisabled" accept=".exe,.pkg,.deb,.msi" @change="changeFileInput($event)"/>
      </div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // Validate and then absorb initial props
    if ((this.initialFileMimeType || this.initialFileMimeType) && !this.initialFileName) {
      throw new Error('<file-upload>: If "initial-file-mime-type" or "initial-file-size" is provided, then "initial-file-name" must also be provided.');
    }
    if (this.mode !== 'image' && this.initialSrc) {
      throw new Error('<file-upload>: Cannot set "initial-src" unless "mode" is "image".');
    }
    if (this.mode === 'image' && this.initialFileName) {
      throw new Error('<file-upload>: Cannot set "initial-file-name" or "initial-file-mime-type" if "mode" is "image".');
    }

    if (this.initialSrc) {
      this.isEmpty = false;
      this.previewSrc = this.initialSrc;
    } else if (this.initialFileName) {
      this.isEmpty = false;
      this.selectedFileName = this.initialFileName;
      this.selectedFileMimeType = this.initialFileMimeType;
      // this.selectedFileIconClass = parasails.util.getMimetypeIconClass(this.initialFileMimeType);
      this.selectedFileSize = this.initialFileSize;
    } else {
      this.isEmpty = true;
    }

    this.isCurrentlyDisabled = !!this.disabled;
  },
  mounted: function (){
    //…
  },
  beforeDestroy: function() {
    //…
  },
  watch: {
    disabled: function(newVal, unusedOldVal) {
      this.isCurrentlyDisabled = !!newVal;
    },
    value: function(newFile, unusedOldVal) {
      this._absorbValue(newFile);
    },
    selectedFileName: function(newFileName) {
      // Emit the update to the parent component
      this.$emit('update:uploadedFilename', newFileName);
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    // FUTURE: add alias that makes clicking on image previewer open the file picker (but only if there is no existing image)
    // FUTURE: if dragging compatible file onto the window, display previewer and button as dropzones
    // FUTURE: think of some way to elegantly deal with paste (probably only for images though)

    changeFileInput: function($event) {
      // Apply spinlock
      if (this.isReadingFileUpload || this.isCurrentlyDisabled) {
        // Note that we can't preventDefault on an input's change event (it's
        // not supported by the browser), so it's possible to end up in a weird
        // situation here where the file input has changed in the DOM, but neither
        // our file previewer nor the harvested form data reflects that.
        // FUTURE: Look for solutions to this edgiest of edge cases
        return;
      }//• (avast)

      var files = $event.target.files;
      if (files.length > 1) {
        throw new Error('<file-upload> component received multiple files!  But at this time, multiple file uploads are not supported, so this should never happen!');
      }

      // Cancelling the native upload window sets `files` to an empty array.
      // So to address this, if you cancel from the native upload window, then
      // we just avast (return early).
      // > In this case, we'll just leave the harvested form data as it was, and
      // > the previewer displaying whatever you had there before.
      var selectedFile = files[0];
      if (!selectedFile) {
        return;
      }//•

      // Even though triggering the input event should fire our watcher, which
      // will do exactly the same thing as this, still go ahead and manually
      // absorb the new file beforehand.
      // > This is just in case the variable provided to v-model/:value is
      // > immutable, such as if it came from `slot-scope` of a parent component.
      this._absorbValue(selectedFile);
      // • FUTURE: make this component smarter so that the browser doesn't have
      //           to double-read the file's bytes in this edge case.  (But this
      //           kind of caching is pretty bug-prone so we should be careful.)

      // Emit an event so the v-model can update with our selected file.
      this.$emit('input', selectedFile);
    },

    //  ╔═╗╦ ╦╔╗ ╦  ╦╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝║ ║╠╩╗║  ║║    ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╚═╝╚═╝╩═╝╩╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝
    doOpenFileBrowser: function() {
      this.$find('[type="file"]').trigger('click');
    },

    //  ╔═╗╦═╗╦╦  ╦╔═╗╔╦╗╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝╠╦╝║╚╗╔╝╠═╣ ║ ║╣   ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╩╚═╩ ╚╝ ╩ ╩ ╩ ╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝
    _absorbValue: function(newFile) {
      // console.log(newFile);
      if (!newFile) {
        this.isEmpty = true;
        this.previewSrc = undefined;
        this.selectedFileName = undefined;
        this.selectedFileMimeType = undefined;
        this.selectedFileSize = undefined;
      } else if (_.isObject(newFile) && newFile.name) {
        // Duck-type File instance

        // Set vm data for the filename and file MIME type in order to render
        // help text / appropriate icon in the DOM.
        this.isEmpty = false;
        this.selectedFileName = newFile.name;
        this.selectedFileName = this.selectedFileName.replace(/^\d{4}\-\d{2}\-\d{2}_/, '');
        this.selectedFileMimeType = newFile.type;
        this.selectedFileIconClass = parasails.util.getMimetypeIconClass(newFile.type);
        this.selectedFileSize = newFile.size;
        // console.log(newFile);
        if (this.mode === 'image') {
          // Set up the file preview for the UI, start reading, and when finished,
          // tear it all down.  (Note that we're using a spinlock just to be safe,
          // in case it turns out we're dealing with a huge file for some reason.)
          this.isReadingFileUpload = true;
          let reader = new FileReader();
          reader.onload = (event)=>{
            this.previewSrc = event.target.result;

            // Unbind this "onload" event & release the lock.
            delete reader.onload;
            this.isReadingFileUpload = false;
          };//œ
          reader.readAsDataURL(newFile);
        }//ﬁ
        // • FUTURE: potentially support changing this "value" to any arbitrary
        //           Blob instance.
        //           (see also FUTURE note above about replacing initial-src,
        //           etc. with tighter v-model integration)
      } else {
        throw new Error(
          'Changing to that value (v-model) for a <file-upload> component from '+
          'the outside is not yet supported!  (Currently, this component only '+
          'supports programmatically setting the value to `null`.)'
        );
      }//ﬁ
      // • FUTURE: potentially also support passing in a string (URL) as some
      //           other prop, then automatically fetching a Blob from it, and
      //           finally emitting an "input" event to set the v-model properly.
    }

  }

});
