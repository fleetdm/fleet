parasails.registerPage('state-of-device-management', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
    barCharts: {
      demographics: {
        gender: [
          {label: 'Male', percent: '52', color: '#A182DF'},
          {label: 'Female', percent: '48', color: '#E59CC4'},
        ],
        age: [
          {label: '18-24', percent: '26.83', color: '#F2A254'},
          {label: '25-34', percent: '28.78', color: '#91D4C7'},
          {label: '35-44', percent: '14.15', color: '#C4C2CE'},
          {label: '45-54', percent: '13.17', color: '#8191E5'},
          {label: '54+', percent: '17.07', color: '#23AD8E'},
        ],
        country: [
          {label: 'United States', percent: '100', color: '#E5D698'},
        ],
        numberOfEmployees: [
          {label: '1001 - 5000', percent: '65', color: '#B99EEF'},
          {label: '5000+', percent: '35', color: '#CB73A3'},
        ],
        role: [
          {label: 'Other', percent: '21.46', color: '#A182DF'},
          {label: 'Security Engineering (EngSec)', percent: '20', color: '#E59CC4'},
          {label: 'Cloud Security', percent: '17.56', color: '#F1AC8C'},
          {label: 'Security Operations', percent: '16.59', color: '#91D4C7'},
          {label: 'Vulnerability Management', percent: '13.17', color: '#C4C2CE'},
          {label: 'Incident Response (CSIRT, DFIR)', percent: '11.22', color: '#8191E5'},
        ],
        industry: [
          {label: 'Healthcare, biotech, pharma, medical', percent: '12.20', color: '#A182DF'},
          {label: 'Marketing, advertising, media', percent: '11.22', color: '#E59CC4'},
          {label: 'Other', percent: '11.22', color: '#F1AC8C'},
          {label: 'Cyber solutions (MSSP, MDR, security vendor)', percent: '10.73', color: '#91D4C7'},
          {label: 'IT, technology, software', percent: '10.73', color: '#C4C2CE'},
          {label: 'State, local, federal government', percent: '10.73', color: '#8191E5'},
          {label: 'Higher education', percent: '9.27', color: '#23AD8E'},
          {label: 'Financial services, insurance, real estate', percent: '8.29', color: '#E5D698'},
          {label: 'Manufacturing, warehouse, logistics', percent: '8.29', color: '#B99EEF'},
          {label: 'Non-profit, K-12 education', percent: '7.32', color: '#CB73A3'},
        ],
        percentWorkingRemote: [
          {label: '100 percent of workspace', percent: '27', color: '#A182DF'},
          {label: '50 percent of workspace', percent: '27', color: '#E59CC4'},
          {label: '25 percent of workspace', percent: '25', color: '#F1AC8C'},
          {label: '0 percent of workspace', percent: '21', color: '#91D4C7'},
        ],
      },
      partOne: {
        totalDevices: [
          {label: '250,000+ total devices', percent: '25.9', color: '#A182DF'},
          {label: '100,000 total devices', percent: '16.6', color: '#E59CC4'},
          {label: '50,000 total devices', percent: '21', color: '#F1AC8C'},
          {label: '10,000 total devices', percent: '15.6', color: '#91D4C7'},
          {label: '1,000 total devices', percent: '21', color: '#C4C2CE'},
        ],
        mdmEnrolledDevices: [
          {label: '25 percent of devices enrolled', percent: '27.8', color: '#A182DF'},
          {label: '50 percent of devices enrolled', percent: '26.8', color: '#E59CC4'},
          {label: '75 percent of devices enrolled', percent: '22.4', color: '#F1AC8C'},
          {label: '100 percent of devices enrolled', percent: '22.9', color: '#91D4C7'},
        ],
        numberOfWorkstations: [
          {label: '1,000 workstations', percent: '20.5', color: '#A182DF'},
          {label: '2,000 workstations', percent: '18.6', color: '#E59CC4'},
          {label: '5,000 workstations', percent: '19', color: '#F1AC8C'},
          {label: '10,000 workstations', percent: '24.9', color: '#91D4C7'},
          {label: '25,000 workstations', percent: '17.6', color: '#C4C2CE'},
        ],
        devicesOnLatestOS: [
          {label: '25 percent of devices', percent: '21.1', color: '#A182DF'},
          {label: '50 percent of devices', percent: '28.57', color: '#E59CC4'},
          {label: '75 percent of devices', percent: '24.84', color: '#F1AC8C'},
          {label: '100 percent of devices', percent: '25.47', color: '#91D4C7'},
        ],
        typesOfDevicesOnLatestOS: [
          {label: 'Network switches and other infrastructure', percent: '36.6', color: '#A182DF'},
          {label: 'Laptops', percent: '30.2', color: '#E59CC4'},
          {label: 'Virtual desktops (VDIs)', percent: '28.3', color: '#F1AC8C'},
          {label: 'Smartphones', percent: '28.3', color: '#91D4C7'},
          {label: 'Servers', percent: '28.3', color: '#C4C2CE'},
          {label: 'Tablets', percent: '26.8', color: '#8191E5'},
          {label: 'Kiosks', percent: '25.9', color: '#23AD8E'},
          {label: 'Linux-based IoT devices', percent: '24.9', color: '#E5D698'},
        ],
        platformsManaged: [
          {label: 'Windows', percent: '39.5', color: '#A182DF'},
          {label: 'Linux', percent: '26.8', color: '#E59CC4'},
          {label: 'macOS', percent: '26.8', color: '#F1AC8C'},
          {label: 'Blackberry OS', percent: '25.4', color: '#91D4C7'},
          {label: 'Chrome OS', percent: '24.4', color: '#C4C2CE'},
          {label: 'Android', percent: '22.4', color: '#8191E5'},
          {label: 'iOS', percent: '18.1', color: '#23AD8E'},
        ],
        platformsWithStruggle: [
          {label: 'Other', percent: '36.1', color: '#A182DF'},
          {label: 'iOS', percent: '28.8', color: '#E59CC4'},
          {label: 'Android', percent: '26.3', color: '#F1AC8C'},
          {label: 'Chrome OS', percent: '25.9', color: '#C4C2CE'},
          {label: 'Linux', percent: '23.4', color: '#91D4C7'},
          {label: 'macOS', percent: '23.4', color: '#8191E5'},
          {label: 'Blackberry OS', percent: '22', color: '#23AD8E'},
        ],
      }
    },
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
  }
});
