/**
 * <multifield>
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

parasails.registerComponent('multifield', {

  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'value',// « 2-way (for v-model)
    'addButtonText',//« Custom text for the "+ Add another" button (optional)
    'inputType',// For customizing the input type.
    'selectOptions',// For select support. An array of objects that have a name and value. ex: [{name: macOS, value: darwin}, {name: Windows, value: windows}]
    'nameAndHostCountSelectOptions',// For nameAndHostCountSelect support. An array of objects that have a name, id, and hostCount. ex: [{"name": "Microsoft office for macOS 16.76","id": 289874,"hostCount": 1}]
    'cloudError',// For highlighting fields, for this to work, the error response needs to return the value of the invalid fields, see api/controllers/update-priority-vulnerabilities to see an example of this.
    'placeholder',// an optional placeholder value for text type inputs
    'disabled',//« for disabling from the outside (e.g. while syncing)
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function () {
    return {
      currentFieldValues: undefined, //« will be initialized to a single-item array in beforeMount
      isCurrentlyDisabled: false, //« controlled by watching `disabled` prop
      optionsForSelect: [],
      inputPlaceholder: '',
      showAddButton: true,
    };
  },
  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div class="multifield-set">
    <div v-if="inputType === 'checkboxes'">
      <div class="d-flex flex-wrap flex-row">
        <div v-for="option in optionsForSelect" :key="option.name" class="form-check mr-3 mb-3">
          <input type="checkbox" :value="option.name" :id="'checkbox-' + option.name" class="form-check-input" @change="inputCheckboxItemField($event)" :checked="_.contains(currentFieldValues, option.name)"/>
          <label :for="'checkbox-' + option.name" class="form-check-label">{{ option.name }}</label>
        </div>
      </div>
    </div>
    <div v-else>
      <div class="multifield-item" v-for="(unused,idx) in currentFieldValues" :key="idx" :role="'item-'+idx">
        <!-- <span class="multifield-item-label">{{idx+1}}.</span> -->
        <slot name="item-field" :item="currentFieldValues[idx]" :do-set="_getCurriedDoSetFn(idx)" :all-items="currentFieldValues" :idx="idx">
          <input type="text" :placeholder="inputPlaceholder" :class="[cloudError && _.contains(cloudError.responseInfo.data, currentFieldValues[idx]) ? 'text-danger is-invalid' : '']" :value.sync="currentFieldValues[idx]" @input="inputDefaultItemField($event, idx)" role="focusable" v-if="!inputType"/>
          <select class="custom-select" :value.sync="currentFieldValues[idx]" @input="inputDefaultItemField($event, idx)" role="focusable" v-else-if="inputType && inputType === 'nameAndHostCountSelect'">
            <option :value="undefined" selected>---</option>
            <option v-for="option in optionsForSelect" :value="option.id">{{option.name}} ({{option.hostCount}} {{option.hostCount > 1 || option.hostCount === 0 ? 'hosts' : 'host'}})</option>
          </select>
          <select class="custom-select" :value.sync="currentFieldValues[idx]" @input="inputDefaultItemField($event, idx)" role="focusable" v-else-if="inputType && inputType === 'select'">
            <option :value="undefined" selected>---</option>
            <option v-for="option in optionsForSelect" :value="option.fleetApid">{{option.name}}</option>
          </select>
          <select class="custom-select" :value.sync="currentFieldValues[idx]" @input="inputTeamSelectItemField($event, idx)" role="focusable" v-else-if="inputType && inputType === 'teamSelect'">
            <option :value="undefined" selected>---</option>
            <option value="allTeams">All teams</option>
            <option v-for="option in optionsForSelect" :value="option.fleetApid">{{option.teamName}}</option>
          </select>
          <select class="custom-select" :disabled="isCurrentlyDisabled" :value.sync="currentFieldValues[idx]" :class="[isCurrentlyDisabled ? 'disabled' : '']" @input="inputTeamSelectItemField($event, idx)" role="focusable" v-else-if="inputType && inputType === 'teamSelectWithNoAllTeamsOption'">
            <option :value="undefined" selected>---</option>
            <option v-for="option in optionsForSelect" :value="option.fleetApid">{{option.teamName}}</option>
          </select>
          <input :type="inputType" :placeholder="inputPlaceholder" :value.sync="currentFieldValues[idx]" @input="inputDefaultItemField($event, idx)" role="focusable" v-else-if="inputType">
        </slot>
        <button class="multifield-item-remove-button" :disabled="isCurrentlyDisabled" type="button" v-if="currentFieldValues.length >= 2" @click="clickRemoveItem(idx)"></button>
        <button class="multifield-item-remove-button" :disabled="isCurrentlyDisabled" type="button" v-else-if="currentFieldValues.length === 1 && currentFieldValues[0] !== undefined" @click="clickResetSingleItem()"></button>
      </div>
      <div class="add-button-wrapper d-flex flex-row justify-content-start" :class="_.all(currentFieldValues, (item)=> item !== undefined) ? '' : 'empty'">
        <a class="add-button" @click="clickAddItem()" v-if="_.all(currentFieldValues, (item)=> item !== undefined) && !isCurrentlyDisabled"><strong>+</strong>&nbsp;&nbsp;{{addButtonText || 'Add another'}}</a>
        <span v-else>&nbsp;</span>
      </div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // Absorb value
    if (this.value !== undefined && !_.isArray(this.value)) {
      throw new Error('In <multifield>, if specified, `v-model`/`:value` must be either an array or `undefined`.  But instead, got: '+this.value);
    }//•
    if (this.value === undefined || _.isEqual(this.value, [])) {
      this.currentFieldValues = [ undefined ];
    } else {
      this.currentFieldValues = _.clone(this.value);
      // ^^ The clone is to prevent entanglement risk.
    }
    if(this.inputType === 'nameAndHostCountSelect') {
      if(!_.isArray(this.selectOptions)){
        throw new Error('Missing selectOptions. When using inputType="nameAndHostCountSelect", an array of selectOptions is required.');
      } else {
        for(let option of this.selectOptions){
          // If we're using inputType="nameAndHostCountSelect", we will validate all options before cloning the object.
          if(!option.id){
            throw new Error(`Option in selectOptions is missing an id. When using inputType="nameAndHostCountSelect", An id property is required for all objects in the selectOptions array. Object missing an id: ${option}`);
          }
          if(!option.name){
            throw new Error(`Option in selectOptions is missing a name. When using inputType="nameAndHostCountSelect", A name property is required for all objects in the selectOptions array. Object missing a name: ${option}`);
          }
          if(option.hostCount === undefined){
            throw new Error(`Option in selectOptions is missing a hostCount. When using inputType="nameAndHostCountSelect", A hostCount property is required for all objects in the selectOptions array. Object missing a hostCount: ${option}`);
          }
        }
        this.optionsForSelect = _.clone(this.selectOptions);
      }
    }
    if(this.inputType === 'teamSelect' || this.inputType === 'teamSelectWithNoAllTeamsOption') {
      if(!_.isArray(this.selectOptions)){
        throw new Error('Missing selectOptions. When using inputType="teamSelect", an array of selectOptions is required.');
      } else {
        for(let option of this.selectOptions){
          // If we're using inputType="nameAndHostCountSelect", we will validate all options before cloning the object.
          if(typeof option.fleetApid !== 'number'){
            throw new Error(`Option in selectOptions is missing an fleetApid. When using inputType="teamSelect", An fleetApid property is required for all objects in the selectOptions array. Object missing an fleetApid: ${option}`);
          }
          if(!option.teamName){
            throw new Error(`Option in selectOptions is missing a teamName. When using inputType="teamSelect", A teamName property is required for all objects in the selectOptions array. Object missing a teamName: ${option}`);
          }
        }
        this.optionsForSelect = _.clone(this.selectOptions);
      }
    }
    if(this.inputType === 'select') {
      if(!_.isArray(this.selectOptions)){
        throw new Error('Missing selectOptions. When using inputType="select", an array of selectOptions is required.');
      } else {
        for(let option of this.selectOptions){
          // If we're using inputType="select", we will validate all options before cloning the object.
          if(!option.value){
            throw new Error(`Option in selectOptions is missing a value. When using inputType="select", A value property is required for all objects in the selectOptions array. Object missing a value. ${option}`);
          }
          if(!option.name){
            throw new Error(`Option in selectOptions is missing a name. When using inputType="select", A name property is required for all objects in the selectOptions array. Object missing a name. ${option}`);
          }
        }
        this.optionsForSelect = _.clone(this.selectOptions);
      }
    }
    if(this.inputType === 'checkboxes') {
      if(!_.isArray(this.selectOptions)){
        throw new Error('Missing selectOptions. When using inputType="select", an array of selectOptions is required.');
      } else {
        for(let option of this.selectOptions){
          // If we're using inputType="select", we will validate all options before cloning the object.
          if(!option.value){
            throw new Error(`Option in selectOptions is missing a value. When using inputType="select", A value property is required for all objects in the selectOptions array. Object missing a value. ${option}`);
          }
          if(!option.name){
            throw new Error(`Option in selectOptions is missing a name. When using inputType="select", A name property is required for all objects in the selectOptions array. Object missing a name. ${option}`);
          }
        }
        this.optionsForSelect = _.clone(this.selectOptions);
        if(this.currentFieldValues === [null]){
          this.currentFieldValues = [];

        }
      }
    }
    if(this.placeholder){
      this.inputPlaceholder = this.placeholder;
    }
  },

  mounted: async function () {

  },

  beforeDestroy: function() {

  },

  watch: {

    value: function(those) {
      // console.log('ran the `value` watcher', those);
      if (those !== undefined && !_.isArray(those)) {
        throw new Error('Cannot programmatically set value for <multifield>: the given value must be an array or `undefined`, but instead got: '+those);
      }
      this.currentFieldValues = those;
    },
    disabled: function(newVal, unusedOldVal) {
      this.isCurrentlyDisabled = !!newVal;
    },

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    //  ╔═╗╦  ╦╔═╗╔╗╔╔╦╗  ╦ ╦╔═╗╔╗╔╔╦╗╦  ╔═╗╦═╗╔═╗
    //  ║╣ ╚╗╔╝║╣ ║║║ ║   ╠═╣╠═╣║║║ ║║║  ║╣ ╠╦╝╚═╗
    //  ╚═╝ ╚╝ ╚═╝╝╚╝ ╩   ╩ ╩╩ ╩╝╚╝═╩╝╩═╝╚═╝╩╚═╚═╝
    inputDefaultItemField: async function($event, idx) {
      var parsedValue = $event.target.value || undefined;
      this.currentFieldValues[idx] = parsedValue;
      await this.forceRender();
      this._handleChangingFieldValues();
    },

    inputTeamSelectItemField: async function($event, idx) {
      var parsedValue = $event.target.value || undefined;
      this.currentFieldValues[idx] = parsedValue;
      if(parsedValue === 'allTeams') {
        this.currentFieldValues = _.pluck(this.optionsForSelect, 'fleetApid');
      }
      await this.forceRender();
      this._handleChangingFieldValues();
    },


    inputTeamSelectWithAllTeamsItemField: async function($event, idx) {
      var parsedValue = $event.target.value || undefined;
      this.currentFieldValues[idx] = parsedValue;
      if(parsedValue === '9999') {
        this.showAddButton = false;
      } else {
        this.showAddButton = true;
      }
      await this.forceRender();
      this._handleChangingFieldValues();
    },

    inputCheckboxItemField: async function($event) {
      let checkboxValue = $event.target.value;
      if($event.target.checked) {
        this.currentFieldValues.push(checkboxValue);
      } else {
        this.currentFieldValues = this.currentFieldValues
        .filter(value => value !== checkboxValue);
      }
      this.currentFieldValues = this.currentFieldValues.filter(value => value !== undefined && value !== null);
      await this.forceRender();
      this._handleChangingFieldValues();
    },

    clickAddItem: async function() {
      this.currentFieldValues.push(undefined);
      await this.forceRender();//«« this is so that the programmatic focusing code below will work

      // Autofocus (but only if we're sure it's going to work)
      var idxToFocus = this.currentFieldValues.length - 1;
      var focalSelector = `[role="item-${idxToFocus}"] [role="focusable"], [role="item-${idxToFocus}"] [focus-first]`;
      var focusableEls = this.$find(focalSelector);
      if (focusableEls.length === 1) {
        this.$focus(focalSelector);
      }
      this.$emit('input', _.clone(this.currentFieldValues));
    },

    clickRemoveItem: async function(idx) {

      this.currentFieldValues.splice(idx, 1);
      this.$emit('input', _.clone(this.currentFieldValues));
      // The _.clone() above is to prevent an entanglement issue caused by
      // emitting the same reference if we were to use this.currentFieldValues
      // directly.

      // Autofocus (but only if we're sure it's going to work)
      var idxToFocus = idx >= this.currentFieldValues.length ? this.currentFieldValues.length - 1 : idx;
      var focalSelector = `[role="item-${idxToFocus}"] [role="focusable"], [role="item-${idxToFocus}"] [focus-first]`;
      var focusableEls = this.$find(focalSelector);
      if (focusableEls.length === 1) {
        this.$focus(focalSelector);
      }
    },
    // To accomodate the requested behavior in wireframes.
    clickResetSingleItem: async function() {
      this.currentFieldValues[0] = undefined;
      this.$emit('input', _.clone(this.currentFieldValues));
      await this.forceRender();
    },

    //  ╔═╗╦ ╦╔╗ ╦  ╦╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝║ ║╠╩╗║  ║║    ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╚═╝╚═╝╩═╝╩╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝

    //…

    //  ╔═╗╦═╗╦╦  ╦╔═╗╔╦╗╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝╠╦╝║╚╗╔╝╠═╣ ║ ║╣   ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╩╚═╩ ╚╝ ╩ ╩ ╩ ╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝
    _getCurriedDoSetFn: function(idx) {
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
        this.currentFieldValues[idx] = newVal;
        await this.forceRender();
        this._handleChangingFieldValues();
      };
    },

    _handleChangingFieldValues: function() {
      // > Note that we do a `_.clone()`.  This is to prevent an entanglement
      // > issue caused by emitting the same reference if we were to simply emit
      // > `this.currentFieldValues` directly.
      this.$emit('input', _.clone(this.currentFieldValues));
      // console.log('emitting in <multifield>…', _.cloneDeep(this.currentFieldValues));
    },

  }

});
