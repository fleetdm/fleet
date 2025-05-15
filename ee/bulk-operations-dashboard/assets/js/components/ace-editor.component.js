/**
 * <ace-editor>
 * -----------------------------------------------------------------------------
 *
 * @type {Component}
 *
 * --- SLOTS: ---
 * @slot item-field
 *       The template to use for each field.
 *       > Also note:
 *       > If this slot contains exactly one element with `role="focusable"` or
 *       > `focus-first`, that element will be focused automatically on add/remove.
 *       @param {Ref} item
 *       @param {Function} doSet
 *       @param {Array} allItems
 *       @param {Number} idx
 *
 * --- EVENTS EMITTED: ---
 * @event input
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('aceEditor', {

  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'value',// « 2-way (for v-model)
    'mode',// For customizing the type of editor
    'maxLines',
    'minLines',
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function () {
    return {
      currentValue: undefined, //« will be initialized to a string in beforeMount
      uniqueId: crypto.randomUUID(),// Used to create a unique ID for the ace editor component.
    };
  },
  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="ace-editor-container">
    <div @input="inputDefaultItemField($event)" @paste="inputDefaultItemField($event)" style="height: 300px;" :id="'editor' + uniqueId" :do-set="_getCurriedDoSetFn()">{{value}}</div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if (this.value === undefined) {
      this.currentValue = undefined;
    } else {
      this.currentValue = _.clone(this.value);
      // ^^ The clone is to prevent entanglement risk.
    }
    if(this.mode){
      if(!['sh', 'fleet', 'powershell'].includes(this.mode)){
        throw new Error(`Invalid mode passed into <ace-editor> component, currently, only 'sh' and 'fleet' are supported.`, this.mode);
      }
    }
  },

  mounted: async function () {
    this._setUpAceEditor(this.mode);
  },

  beforeDestroy: function() {

  },

  watch: {

    currentValue: function() {
      this.currentValue = ace.edit('editor'+this.uniqueId).getValue();
    }

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    //  ╔═╗╦  ╦╔═╗╔╗╔╔╦╗  ╦ ╦╔═╗╔╗╔╔╦╗╦  ╔═╗╦═╗╔═╗
    //  ║╣ ╚╗╔╝║╣ ║║║ ║   ╠═╣╠═╣║║║ ║║║  ║╣ ╠╦╝╚═╗
    //  ╚═╝ ╚╝ ╚═╝╝╚╝ ╩   ╩ ╩╩ ╩╝╚╝═╩╝╩═╝╚═╝╩╚═╚═╝
    inputDefaultItemField: async function($event) {
      var parsedValue = $event.target.value || undefined;
      this.currentValue = parsedValue;
      await this.forceRender();
      this._handleChangingFieldValues();
    },


    //  ╔═╗╦ ╦╔╗ ╦  ╦╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝║ ║╠╩╗║  ║║    ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╚═╝╚═╝╩═╝╩╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝

    //…

    //  ╔═╗╦═╗╦╦  ╦╔═╗╔╦╗╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝╠╦╝║╚╗╔╝╠═╣ ║ ║╣   ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╩╚═╩ ╚╝ ╩ ╩ ╩ ╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝
    _getCurriedDoSetFn: function() {
      return async (newVal)=>{
        // Note that it is the responsibility of the userland contents of the
        // slot to make sure this incoming value is proper.  For example, if
        // the slot contains an `<input>`, then when invoking doSet, you should
        // do so like:
        // ```
        // <input :value="item" @input="doSet($event.target.value||undefined)"/>
        // ```
        //
        // The `||undefined` is because otherwise, you get `null`, and you
        // probably want blank fields to be treated as undefined so we can
        // automatically splice them out of the array before emitting our input
        // event.
        //
        // The reason this is left as a userland concern is because the `null`
        // value itself, just like `''`, `0`, `false`, `NaN` or other similar
        // values, is technically a valid thing that might be relevant under
        // unusual circumstances.
        // - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
        // console.log('FIRED DOSET for idx',idx,'and newVal', newVal);
        this.currentValue = newVal;
        await this.forceRender();
        this._handleChangingFieldValues();
      };
    },

    _handleChangingFieldValues: function() {
      // > Note that we do a `_.clone()`.  This is to prevent an entanglement
      // > issue caused by emitting the same reference if we were to simply emit
      // > `this.currentValue` directly.
      this.$emit('input', _.clone(this.currentValue));
      // console.log('emitting in <multifield>…', _.cloneDeep(this.currentValue));
    },

    _setUpAceEditor: function(mode) {
      var editor = ace.edit('editor'+ this.uniqueId);
      ace.config.setModuleUrl('ace/mode/fleet', '/dependencies/src-min/mode-fleet.js');
      editor.setTheme('ace/theme/fleet');
      editor.session.setMode('ace/mode/'+mode);
      editor.setOptions({
        minLines: this.minLines ? this.minLines : 4 ,
        maxLines:  this.maxLines ? this.maxLines : 11 ,
      });
    },

  }

});
