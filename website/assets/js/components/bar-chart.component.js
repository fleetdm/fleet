/**
 * <ajax-button>
 * -----------------------------------------------------------------------------
 * A button with a built-in loading spinner.
 *
 * @type {Component}
 *
 * @event click   [emitted when clicked]
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('barChart', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'chartData',
    'title',
    'type',
    'maxRange',
    'minRange',
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    let range = this.maxRange - this.minRange;
    let incrementBy = undefined;
    if (range >= 20) {
      incrementBy = 5;
    } else if(range > 10) {
      incrementBy = 2;
    } else {
      incrementBy = 1;
    }
    return {
      range,
      incrementBy,
      chartRange: undefined,
      //…
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div v-if="type === 'stacked'">
      <span purpose="title">{{title}}</span>
      <div purpose="chart" class="d-flex">
        <span v-for="item in chartData" :style="'flex-basis: '+item.percent+'%; background-color: '+item.color+';'">
        </span>
      </div>
      <div purpose="chart-labels" class="d-flex" :class="[chartData.length === 1 ? 'justify-content-around' : '']">
        <span v-for="item in chartData" :style="'flex-basis: '+item.percent+'%;'">
          <span purpose="label"><strong>{{item.percent}}% </strong>{{item.label}}</span>
        </span>
      </div>
    </div>
    <div v-else-if="type === 'divided'">
      <span purpose="title">{{title}}</span>
      <div class="d-flex flex-column pb-3" v-for="item in chartData">
        <div purpose="chart" class="d-flex">
        <span purpose="chart-fill" :style="'flex-basis: '+((item.percent - minRange) / range * 100)+'%; background-color: '+item.color+';'">
        </span>
      </div>
        <span purpose="label"><strong>{{item.percent}}% </strong>{{item.label}}</span>
      </div>
      <div purpose="range" class="pt-3 d-flex flex-row justify-content-between">
        <span class="d-flex" v-for="item in chartRange">
          {{item}}%
        </span>
      </div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
  },
  mounted: async function() {
    if(this.type === 'divided') {
      if(this.maxRange && this.minRange){
        this.range = this.maxRange - this.minRange;
        if(!this.incrementBy){
          this.chartRange = Array.from({length: (this.range + 1)}, (_, i) => i + parseInt(this.minRange));
        } else {
          this.chartRange = Array.from({length: ((this.range)/this.incrementBy + 1)}, (_, i) => (i * this.incrementBy) + parseInt(this.minRange));
        }
      }
    }


  },
  beforeDestroy: function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

  }
});
