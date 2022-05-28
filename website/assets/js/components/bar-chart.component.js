/**
 * <bar-chart>
 * -----------------------------------------------------------------------------
 * A button with a built-in loading spinner.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('barChart', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'type', // Required: The type of bar chart to display. either 'stacked' (values are combined onto a single line) or 'divided' (each value is displayed as a seperate line)
    'chartData', // Required: an array of objects, each containing a 'label', 'percent', and 'color'
    'title', // Required: the title of the chart
    'subtitle', // Optional: if provided, a subtitle will be added the chart
    'maxRange', // Required for 'divided' type, the lowest number for the scale to display
    'minRange', // Required for 'divided' type, the highest number for the scale to display
    'incrementScaleBy', // Optional: if provided the scale will increment by this number, otherwise the scale will increment by 1, 2, 5, or 10
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    let range = this.maxRange - this.minRange;
    let incrementBy = undefined;

    // Determine how the range should increment, if a number was provided as a prop (`incrementScaleBy`),
    // we'll use that, otherwise we'll set this value based on the range of the bar chart
    if(this.incrementScaleBy){
      incrementBy = this.incrementScaleBy;
    } else if (range >= 50) {
      incrementBy = 10;
    } else if (range >= 20) {
      incrementBy = 5;
    } else if(range > 10) {
      incrementBy = 2;
    } else {
      incrementBy = 1;
    }

    return {
      range,
      incrementBy,
      chartScale: undefined,
      //…
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div v-if="type === 'stacked'">
      <div purpose="stacked-title">
        {{title}}
        <span purpose="subtitle" v-if="this.subtitle">{{subtitle}}</span>
      </div>
      <div purpose="chart" class="d-flex">
        <span v-for="item in chartData" :style="'flex-basis: '+item.percent+'%; background-color: '+item.color+';'">
        </span>
      </div>
      <div purpose="chart-labels" class="d-flex" :class="[chartData.length === 1 ? 'justify-content-around' : '']">
        <span purpose="label-container" v-for="item in chartData" :style="'flex-basis: '+item.percent+'%;'">
          <span purpose="label"><strong>{{item.percent}}% </strong><span>{{item.label}}</span></span>
        </span>
      </div>
    </div>
    <div v-else-if="type === 'divided'">
      <div purpose="divided-title">
        {{title}}
        <span purpose="subtitle" v-if="this.subtitle">{{subtitle}}</span>
      </div>
      <div class="d-flex flex-column pb-3" v-for="item in chartData">
        <div purpose="chart" class="d-flex">
        <span purpose="chart-fill" :style="'flex-basis: '+((item.percent - minRange) / range * 100)+'%; background-color: '+item.color+';'">
        </span>
      </div>
        <span purpose="label"><strong>{{item.percent}}% </strong>{{item.label}}</span>
      </div>
      <div purpose="range" class="d-flex flex-row align-self-end justify-content-between">
        <span v-for="value in chartScale">
          {{value}}%
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

    if(this.type === undefined){
      throw new Error('Incomplete usage of <bar-chart>:  Please provide a `type`, either "divided" or "stacked". For example: `<bar-chart type="divided">`');
    } else if (this.type !== 'divided' && this.type !== 'stacked'){
      throw new Error('<bar-chart> received an invalid `type`. `type` should be either "divided" or "stacked"');
    }
    if(this.chartData === undefined){
      throw new Error('Incomplete usage of <bar-chart>:  Please provide an array of objects as `chartData`. For example: `<bar-chart :chart-data="barCharts.demographics">`');
    } else if (!_.isArray(this.chartData)){
      throw new Error('<bar-chart> received an invalid `chartData`. `chartData` should be an array of objects. Each object should containing a `label` (string), `percent` (string), and `color` (string).');
    }
    if(this.title === undefined){
      throw new Error('Incomplete usage of <bar-chart>:  Please provide a `title`. For example: `<bar-chart title="My great chart">`');
    }
    if(this.type === 'divided') {
      if(this.maxRange === undefined) {
        throw new Error('Incomplete usage of <bar-chart>: When using the `divided` type a `maxRange` is required. For example: `<bar-chart max-range="30">`');
      }
      if(this.minRange === undefined) {
        throw new Error('Incomplete usage of <bar-chart>: When using the `divided` type a `minRange` is required. For example: `<bar-chart min-range="10">`');
      }
    }

    // Adjusting the range for divided bar charts
    if(this.type === 'divided') {
      if(this.maxRange && this.minRange){
        this.chartScale = Array.from({length: ((this.range)/this.incrementBy + 1)}, (_, i) => (i * this.incrementBy) + parseInt(this.minRange));
      }
    }
  },
  watch: {
    type: function(unused) { throw new Error('Changes to `type` are not currently supported in <bar-chart>!'); },
    chartData: function(unused) { throw new Error('Changes to `chartData` are not currently supported in <bar-chart>!'); },
    title: function(unused) { throw new Error('Changes to `title` are not currently supported in <bar-chart>!'); },
    subtitle: function(unused) { throw new Error('Changes to `subtitle` are not currently supported in <bar-chart>!'); },
    maxRange: function(unused) { throw new Error('Changes to `maxRange` are not currently supported in <bar-chart>!'); },
    minRange: function(unused) { throw new Error('Changes to `minRange` are not currently supported in <bar-chart>!'); },
    incrementScaleBy: function(unused) { throw new Error('Changes to `incrementScaleBy` are not currently supported in <bar-chart>!'); },
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
