parasails.registerPage('state-of-device-management', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {

    pieCharts: {
      cloudMDM:{
        elementID: 'cloud-mdm-chart',
        legendPosition: 'bottom',
        data: {
          labels: ['Cloud solution', 'Self-managed'],
          datasets: [{
            label: 'percent',
            data: [57.6, 42.4],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmCharges: {
        elementID: 'mdm-charges-chart',
        legendPosition: 'bottom',
        data: {
          labels: ['MDM charges per device', 'MDM charges per user'],
          datasets: [{
            label: 'percent',
            data: [50.38, 49.62],
            backgroundColor: ['#F2A254', '#91D4C7'],
          }]
        },
      },
      mdmInvolvement: {
        elementID: 'mdm-involvement-chart',
        data: {
          labels: ['Yes', 'No'],
          datasets: [{
            label: 'percent',
            data: [48.3, 51.7],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmDeploymentTime: {
        elementID: 'mdm-deployment-time-chart',
        data: {
          labels: ['<4 weeks', '1-3 months', '3-6 months', '6-9 months', '9-12 months', '12+ months'],
          datasets: [{
            label: 'percent',
            data: [18.69, 15.66, 13.13, 16.16, 14.14, 22.22],
            backgroundColor: ['#A182DF', '#E59CC4','#F2A254', '#91D4C7','#C4C2CE','#8191E5'],
          }]
        },
      },
      mdmVisibility: {
        elementID: 'mdm-visibility-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [46.83, 53.17],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmLaptopsAndServers: {
        elementID: 'mdm-laptops-and-servers-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [48.8, 51.2],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmInventory: {
        elementID: 'mdm-inventory-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [49.8, 50.2],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmIncidentResponse: {
        elementID: 'mdm-incident-response-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [51.7, 48.3],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmRealtimeVisibility: {
        elementID: 'mdm-realtime-visibility-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [54.6, 45.4],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmComplianceEnforcement: {
        elementID: 'mdm-compliance-enforcement-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [48.8, 51.2],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmWorkflowAutomation: {
        elementID: 'mdm-workflow-automation-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [54, 46],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
      mdmSelfService: {
        elementID: 'mdm-self-service-chart',
        data: {
          labels: ['Yes', 'No',],
          datasets: [{
            label: 'percent',
            data: [48.8, 51.2],
            backgroundColor: ['#A182DF', '#E59CC4'],
          }]
        },
      },
    },
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
          {label: '100 percent of workforce', percent: '27', color: '#A182DF'},
          {label: '50 percent of workforce', percent: '27', color: '#E59CC4'},
          {label: '25 percent of workforce', percent: '25', color: '#F1AC8C'},
          {label: '0 percent of workforce', percent: '21', color: '#91D4C7'},
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
      },
      partTwo: {
        deviceManagementStrategy: [
          {label: 'Having a documented BTOD policy in place', percent: '31.7', color: '#A182DF'},
          {label: 'Reporting how quickly vulnerable software is patched', percent: '30.2', color: '#E59CC4'},
          {label: 'Providing a self-service experience', percent: '28.3', color: '#F1AC8C'},
          {label: 'Focusing on a seamless end-user experience ', percent: '27.3', color: '#91D4C7'},
          {label: 'Having a documented AUP (Acceptable Use Policy)', percent: '25.9', color: '#C4C2CE'},
          {label: 'Tracking ownership, location, use, and other aspects of devices', percent: '25.9', color: '#8191E5'},
          {label: 'Enabling risk-based policies', percent: '24.9', color: '#23AD8E'},
          {label: 'Adopting passwordless authentication', percent: '23.9', color: '#E5D698'},
          {label: 'Allowing scalable updates to numerous devices and apps', percent: '22.4', color: '#B99EEF'},
          {label: 'Measuring point-in-time compliance across all devices', percent: '30.7', color: '#CB73A3'},
        ],
        topChallenges: [
          {label: 'Verifying compliance across devices', percent: '22.9', color: '#A182DF'},
          {label: 'Getting all devices enrolled', percent: '21', color: '#E59CC4'},
          {label: 'Maintaining accurate visibility across all devices', percent: '20.5', color: '#F1AC8C'},
          {label: 'Balancing end-user experience with security', percent: '19', color: '#91D4C7'},
          {label: 'Investigating devices in real time', percent: '16.6', color: '#C4C2CE'},
        ],
      },
      partThree: {
        mdmInvestmentTriggers: [
          {label: 'The shift to remote work', percent: '16.7', color: '#A182DF'},
          {label: 'An upcoming compliance audit or certification', percent: '16.1', color: '#E59CC4'},
          {label: 'More control over onboarding/enrollment experience', percent: '15.6', color: '#F1AC8C'},
          {label: 'More visibility about the device and its installed software', percent: '15.1', color: '#91D4C7'},
          {label: 'No other way to limit configuration options available to end users', percent: '14', color: '#C4C2CE'},
          {label: 'Need to keep an accurate inventory of company devices', percent: '11.8', color: '#8191E5'},
          {label: 'Required by a customer contract', percent: '10.8', color: '#23AD8E'},
        ],
        challengesEncountered: [
          {label: 'Too complicated to configure and understand', percent: '36.1', color: '#A182DF'},
          {label: 'Confusing or limited documentation', percent: '34.2', color: '#E59CC4'},
          {label: 'Integrating with single sign-on (Okta, SAML, etc.)', percent: '32.7', color: '#F1AC8C'},
          {label: 'Hard to get support', percent: '32.7', color: '#91D4C7'},
          {label: 'Unable to automate everything we want with the API', percent: '32.2', color: '#C4C2CE'},
          {label: 'Didn’t know where to start', percent: '30.7', color: '#8191E5'},
          {label: 'Migrating devices enrolled in a previous MDM', percent: '30.2', color: '#23AD8E'},
        ],
      },
      partFive: {
        expectedBudgetChange: [
          {label: 'Budget will increase', percent: '32.5', color: '#A182DF'},
          {label: 'Budget will decrease', percent: '32.5', color: '#E59CC4'},
          {label: 'Budget will stay the same', percent: '35.1', color: '#F1AC8C'},
        ],
        mostWantedFeatures: [
          {label: 'Cloud-hosted by vendor', percent: '28.3', color: '#A182DF'},
          {label: 'Built-in security controls with sensible defaults', percent: '28.3', color: '#E59CC4'},
          {label: 'Inspectable, modifiable, open source code', percent: '26.3', color: '#F1AC8C'},
          {label: 'Initial setup of software or tools at onboarding', percent: '25.9', color: '#91D4C7'},
          {label: 'Real-time visibility of every device, less than a minute old', percent: '25.9', color: '#C4C2CE'},
          {label: 'Collecting security data from enrolled devices', percent: '25.9', color: '#8191E5'},
          {label: 'GitOps workflows for configuration', percent: '24.4', color: '#23AD8E'},
          {label: 'Remotely locking or wiping devices', percent: '23.4', color: '#E5D698'},
          {label: 'Integration with single sign-on (Okta, etc.)', percent: '23.4', color: '#B99EEF'},
          {label: '24/7 support', percent: '23.4', color: '#CB73A3'},
          {label: 'Developer-friendly API and webhooks', percent: '22.9', color: '#E3B6A0'},
          {label: 'Device posture information available in API or integrations ("zero trust")', percent: '22.9', color: '#91D49C'},
          {label: 'Ability to know what apps, packages, or browser extensions are installed', percent: '22.9', color: '#A5A1B7'},
          {label: 'Ability to patch operating systems', percent: '22.4', color: '#82A8D3'},
          {label: 'Self-managed (hosted by company)', percent: '22.4', color: '#23ADA5'},
          {label: 'Ability to patch third party apps, packages, browser plugins', percent: '22', color: '#F1CD98'},
          {label: 'Device types supported', percent: '22', color: '#8972B8'},
          {label: 'Ability to auto-detect new vulnerabilities in packages, apps, browser plugins', percent: '20', color: '#FC838A'},
          {label: 'Ability to limit OS configuration available to end users ("guest users," etc.)', percent: '19.5', color: '#F2D68E'},
          {label: 'Self-service IT support', percent: '17.6', color: '#9DC78E'},
        ],
        deviceManagementPriorities: [
          {label: 'Device security such as multi-factor authentication (MFA) at login', percent: '18.1', color: '#A182DF'},
          {label: 'Zero-touch enrollment', percent: '14.2', color: '#E59CC4'},
          {label: 'Patching third-party applications and packages', percent: '13.2', color: '#F1AC8C'},
          {label: 'Supporting fully remote teams', percent: '12.7', color: '#91D4C7'},
          {label: 'Patching operating systems', percent: '11.7', color: '#C4C2CE'},
          {label: 'Improving overall visibility', percent: '11.2', color: '#8191E5'},
          {label: 'Adopting a self-service IT approach', percent: '10.2', color: '#23AD8E'},
          {label: 'Blocking or allowing SSO (single sign-on) based on device state', percent: '8.8', color: '#E5D698'},
        ],
      },
    },
    chartsDrawnOnPage: [],
    redrawnCharts: [],
    scrollDistance: 0,

  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function() {
    this.drawChartsOnPage();
    window.addEventListener('resize', this.updateChartsOnPage);
    window.addEventListener('scroll', this.scrollSideNavigationWithHeader);
    this.updateChartsOnPage();
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    scrollSideNavigationWithHeader: function () {
      var navBar = document.querySelector('div[purpose="report-sidebar"]');
      var scrollTop = window.pageYOffset || document.documentElement.scrollTop;
      if(navBar) {
        if (scrollTop > this.scrollDistance && scrollTop > window.innerHeight * 1.5) {
          navBar.classList.add('header-hidden');
        } else {
          if(scrollTop === 0) {
            navBar.classList.remove('header-hidden');
          } else {
            navBar.classList.remove('header-hidden');
          }
        }
      }
      this.scrollDistance = scrollTop;
    },

    drawChartsOnPage: function() {
      for(let index in this.pieCharts) {
        const ctx = this.pieCharts[index].elementID;
        let defaultConfig = [{
          color: '#000000',
          borderWidth: 0,
          borderColor: 'rgba(0,0,0,0)',
        }];
        // cloning this chart's data object to apply the standard configuration options
        let clonedChartData = _.clone(this.pieCharts[index].data);
        _.merge(clonedChartData.datasets, defaultConfig);
        // setting a flag based on wether or not this chart has a legend on the bottom. If the legend is on the bottom, we'll adjust the aspect ratio and padding of the chart.
        let chartHasLegendOnBottom = this.pieCharts[index].legendPosition === 'bottom';
        let chart = new Chart(ctx, {
          type: 'doughnut',
          data: clonedChartData,
          options: {
            cutoutPercentage: 40,
            responsive: true,
            aspectRatio: chartHasLegendOnBottom ? .8 : 1.25,
            maintainAspectRatio: false,
            layout: {
              padding: {
                left: 0,
                bottom: 0,
                top: 16,
                // setting right padding if a legend postion was specified
                right: chartHasLegendOnBottom ? 0 : 50,
              },
            },
            legend: {
              fullWidth: false,
              position: chartHasLegendOnBottom ? 'bottom' : 'right',
              // removing the default onClick event from the chart's legend
              onClick: ()=>{return;},
              labels: {
                padding: 16,
                generateLabels: (chart) => {
                  const datasets = chart.data.datasets;
                  return datasets[0].data.map((data, i) => ({
                    text: `${chart.data.labels[i]} (${data}%)`,
                    fillStyle: datasets[0].backgroundColor[i],
                    pointStyle: 'rectRounded',
                    lineWidth: 0,
                  }));
                },
                fontColor: '#000',
                usePointStyle: true,
                fontSize: 14,
                fontFamily: 'Inter',
              }
            }
          }
        });
        this.chartsDrawnOnPage.push(chart);
      }
    },

    updateChartsOnPage: async function() {

      if(this.redrawnCharts.length < 1) {
        // Iterating through charts drawn on the page. If the window width is below 768px, we'll change the configuration and update the charts.
        for(let index in this.chartsDrawnOnPage) {
          // If a bottom legend position was specified, we'll ignore it.
          let chartHasLegendOnBottomAtAllWidths = this.chartsDrawnOnPage[index].aspectRatio === 0.8;
          if(window.innerWidth < 768 && !chartHasLegendOnBottomAtAllWidths){
            this.redrawnCharts.push(this.chartsDrawnOnPage[index]);
            this.chartsDrawnOnPage[index].aspectRatio = 1;
            this.chartsDrawnOnPage[index].options.legend.position = 'bottom';
            this.chartsDrawnOnPage[index].options.legend.fontSize = 13;
            this.chartsDrawnOnPage[index].options.legend.labels.padding = 8;
            this.chartsDrawnOnPage[index].options.layout.padding.right = 0;
            this.chartsDrawnOnPage[index].update();
            this.chartsDrawnOnPage[index].resize();
          } else if(!chartHasLegendOnBottomAtAllWidths) {
            this.redrawnCharts.push(this.chartsDrawnOnPage[index]);
            this.chartsDrawnOnPage[index].aspectRatio = 2;
            this.chartsDrawnOnPage[index].options.layout.padding.right = 50;
            this.chartsDrawnOnPage[index].options.legend.position = 'right';
            this.chartsDrawnOnPage[index].update();
            this.chartsDrawnOnPage[index].resize();
          }
        }
      } else {
        // Iterating through the charts that have been redrawn and changing them back to their original configuration.
        for(let index in this.redrawnCharts) {
          if(window.innerWidth < 768){
            this.redrawnCharts[index].aspectRatio = 1;
            this.redrawnCharts[index].canvas.attributes.style.height = '340px';
            this.redrawnCharts[index].options.legend.position = 'bottom';
            this.redrawnCharts[index].options.legend.fontSize = 13;
            this.redrawnCharts[index].options.legend.labels.padding = 8;
            this.redrawnCharts[index].options.layout.padding.right = 0;
            this.redrawnCharts[index].update();
            this.redrawnCharts[index].resize();
          } else {
            this.redrawnCharts[index].aspectRatio = 1.25;
            this.redrawnCharts[index].options.layout.padding.right = 50;
            this.redrawnCharts[index].options.legend.position = 'right';
            this.redrawnCharts[index].options.legend.fontSize = 14;
            this.redrawnCharts[index].options.legend.labels.padding = 16;
            this.redrawnCharts[index].update();
            this.redrawnCharts[index].resize();
          }
        }
      }
    },

  }
});
//
