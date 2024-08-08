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
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="clearfix" :class="[mode === 'image' ? 'image-mode' : 'file-mode', isCurrentlyDisabled ? 'disabled' : '']">
    <div class="image-preview" v-if="mode === 'image'">
      <div class="image-preview-field" :class="[isEmpty ? 'empty' : '']" :style="{ backgroundImage: (isEmpty ? 'url('+(placeholderImageSrc||'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEYAAABGCAYAAABxLuKEAAAAAXNSR0IArs4c6QAABU1JREFUeAHtnE1oXFUUgDOT+aFdaGKsJg3FKIggiNOioLjo0E0XipQWC1XQlEB+oMS2tFjbzYh/lJb+xNj8kQSpoMzC1oXgrg0uXPYHBKEI6aYEkdC6KMnEJH63zAzpY+5w7r1vknnMvfB47513zrnnfHN/zpsJiU1MTDy7srLyTVNT09scmzkauT1cXV39IRaLDSSWl5e/42J3I9NYk/tmWPRwX4hzsWvNA38JAUZNd5xz0tN4nACDZZMC41sFAh5MBShK5MFowCQ0crUAXenv79+rex51+ejo6DHWkjO6PPyI0ZDxYDwYDQGNWLvGaPQfiaenp1sWFxc/4mYP8zTDuYXjPuvSTc5XWZsuco50MwYzPj6eAcoVgHQFMm9BlkWWHRsb6+b6YG9vrwIVyWa0xoyMjGQZFTcqQAkmn0HvmoIYfBCVezEYNX0AMm2QWAtwTPQNXNdeVQymUChcEIyUYMQZplUuKIzCvRgMyeyxTChrabehZiIwahoR5ZOWke60tNtQMxGYhYUFp0U0iouwCExzc/N9l48vitu2CIxLYuxMt1ygKtvJycmtqlRgIX/F1ZfUXgRGOSPBn6VOA3qzgXujW4DsX1pauhuPx69heBs4h4wcWCqLwRBYzqKPBwA9bGH3yGR4eLiNfscoE8oVOv7O85XBq7Y+pXZiMMXp9JnUcVGve2BgYNbQpqyeSCTOcqN2xHIrQrpQFtToQgxG9d/X15fjJIHzAL0j6F/lbNUYFW8CobuSMfIsO907lZ6FJTMCozpVcAhsO5czmiBm+AEvg571p5rL5dTUqfo6QR/n8vl8syYGZ3F57pp4Kk6rrLJRuwWnLo5Zps11zs6to6PjU5y8VM0RH86L8/PzfehcqqZn+8wKzNrOwoJR8slPxs/z6+gpEi+JtGd0Ph8aGvp+cHDwX62S5QPjqWTZj9gMKFMknBYaPJVKpU4KdY3U6goM0/IDoGRNMkD/qCoATWwkunUDpvh9z0VJ0AGdJAWg2tZDbXUDhu97TvPpt9lkh90BKuIdNrY6m7oAo2oWAuzVBSmRUxF/K9GT6mw4GEnNIkmGUfMGRd8+ia5EZ8PBtLe3f01SVWsWSSJKh6LvLFMqlD9rca5jgkEzLV4j0Y+R7+dIMcT/5n6C955LPT0990r6jJQ4hdwx7tURSqOfLhypt+/zHE4tVDB8WkeIRv1QXi7VCfYZ7k+xc3zC8zzXd4CVQv4+189xhNrwnaPom3Qt+kIDQ9Kq0PpSlyUgVF8KRhPXOjVnOb6fSKfTORwddXEWyhrDovcuQXzhEkjItocoFrtcfDqDYU15mUXvR4Ko3TAwzzDJF1xORZ8TmKmpqS3E/CvDd5N57DW32Fesj6w6sgbDApdmQf0FKNusel4HI2Ibtu3GGgwL3GU6fd2243Wy28GoOWDTlxUYdqATdPaeTYfrbcOosSr6jMEUd6Cv1jtBh/62UtsYb91GYOp0B5IwO67WRIliSUcMRpXwGP1UpztQKZ+KZ2JuSyaT6g1e3MRgOjs736KDUF72xNGFqEhd85+JOzEYirh7zFUj5yaB1FKXuH9rbW393aQP8bsSvxP9pdYYOtkbpelEvHfn5uYu85ekyzUBo5zi/A6n0yYdRFVXPJWimqBt3DGKtVWN8T/I/9A8i7yYKbaNJeEFXSLV1pinMdqpM4y6HChVU/BTSYPHg/FgNAQ0Yj9iPBgNAY3YjxgPRkNAI/YjxoPRENCI/YjxYDQENGI/YqqAKWieNbK4oEbM9UYmUCl3vpKYifMHPQd5+GclhQaUqf9RledvjT9swNxlKf8PY15tW38ii5sAAAAASUVORK5CYII=')+')' : 'none') }">
        <img alt="preview of the image to be uploaded" v-if="previewSrc" :src="previewSrc"/>
      </div>
    </div>
    <span class="file-metadata-preview" v-else-if="selectedFileName">
      <i class="selected-file-mime-type fa" :class="selectedFileIconClass"/>
      &nbsp;<span class="selected-file-name">{{selectedFileName}}</span>
      <span class="selected-file-size" v-if="selectedFileSize && selectedFileSize > 1000000000">&nbsp;{{selectedFileSize / 1000000000 | round(1)}} GB</span>
      <span class="selected-file-size" v-else-if="selectedFileSize && selectedFileSize > 1000000">&nbsp;{{selectedFileSize / 1000000 | round(1)}} MB</span>
      <span class="selected-file-size" v-else-if="selectedFileSize && selectedFileSize > 1000">&nbsp;{{selectedFileSize / 1000 | round}} KB</span>
      <span class="selected-file-size" v-else-if="selectedFileSize">&nbsp;{{selectedFileSize}} B</span>
    </span>
    <div class="btn-and-tips-if-relevant">
      <slot name="image-upload-instructions">
        <p class="text-muted" v-if="mode === 'image'">
          <span>
            <strong v-if="isEmpty">Please select an image.</strong>
            <strong v-else>Here is your image.</strong>
          </span>
          <br/>
          <span >For best results, choose a .png, .jpg, or .gif file smaller than 3 MB.</span>
          <!-- 3 MB is roughly the upper limit of how big images are when captured from modern mobile devices. -->
        </p>
      </slot>
      <span class="file-upload-button" :class="[buttonClass || 'btn btn-outline-primary', isEmpty ? 'no-file-selected' : 'file-selected']">
        <span class="button-text" v-if="isEmpty">Choose {{mode === 'image' ? 'image' : 'a file'}}</span>
        <span class="button-text" v-else>Change {{mode === 'image' ? 'image' : 'file'}}</span>
        <input type="file" class="file-input" :disabled="isCurrentlyDisabled" :accept="mode === 'image' ? 'image/*' : ''" @change="changeFileInput($event)"/>
      </span>
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
        console.log(newFile);
      } else if (_.isObject(newFile) && newFile.name) {
        // Duck-type File instance

        // Set vm data for the filename and file MIME type in order to render
        // help text / appropriate icon in the DOM.
        this.isEmpty = false;
        this.selectedFileName = newFile.name;
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
