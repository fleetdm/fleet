parasails.registerPage('capex-savings', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    costPerDevice: 3300,
    currentCycleYears: 3,
    deviceCount: 2000,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
  },

  //  ╔═╗╔═╗╔╦╗╔═╗╦ ╦╔╦╗╔═╗╔╦╗
  //  ║  ║ ║║║║╠═╝║ ║ ║ ║╣  ║║
  //  ╚═╝╚═╝╩ ╩╩  ╚═╝ ╩ ╚═╝═╩╝
  computed: {
    // Slider label for the cost input (e.g. "$3,300").
    formattedCostPerDevice: function() {
      return '$' + Number(this.costPerDevice).toLocaleString('en-US');
    },
    calculatorResult: function() {
      let cost = Number(this.costPerDevice);
      let cycleYears = Number(this.currentCycleYears);
      let devices = Math.max(1, Number(this.deviceCount) || 0);
      // Monthly cost on the current cycle vs. the same cycle + 1 year.
      let monthlyCurrent = cost / (cycleYears * 12);
      let monthlyExtended = cost / ((cycleYears + 1) * 12);
      let savingsPerDeviceMonthly = monthlyCurrent - monthlyExtended;
      let fleetSavingsYearly = savingsPerDeviceMonthly * devices * 12;
      return '$' + Math.round(fleetSavingsYearly).toLocaleString('en-US');
    },
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
  }
});
