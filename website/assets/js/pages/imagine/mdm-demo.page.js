parasails.registerPage('mdm-demo', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    demoStage: 1,
    counter: {
      hostOne: 0,
      hostTwo: 0,
      hostThree: 0,
      gitOps: 0,
    },
    dragElements:{
      fileOne: undefined,
      fileTwo: undefined,
      fileThree: undefined,
    },
    dropTargets: {
      hostOne: undefined,
      hostTwo: undefined,
      hostThree: undefined,
    },
    gameDurationInSeconds: 25,
    timeLeft: 25,
    finalFinishingTime: undefined,
    showDeployingMsg: false,
    formData: {},
    formErrors: {},
    formRules: {},
    syncing: false,
    cloudError: false,
    showSuccessMessage: false,
    showStageFourDeployButton: false,
    showStageFourInstructions: true,
    stageFourDeployedLinuxFilesFinalPositions:[
      { top: '36px', right: '120px'},
      { top: '98px', right: '39px'},
      { top: '148px', right: '152px'},
      { top: '186px', right: '300px'},
      { top: '71px', right: '246px'},
    ],
    stageFourDeployedWindowsFilesFinalPositions:[
      { top: '300px', right: '296px'},
      { top: '326px', right: '203px'},
      { top: '239px', right: '178px'},
      { top: '238px', right: '60px'},
      { top: '338px', right: '13px'},
      { top: '391px', right: '144px'},
    ],
    stageFourDeployedMacFilesFinalPositions:[
      { top: '566px', right: '274px'},
      { top: '414px', right: '292px'},
      { top: '489px', right: '186px'},
      { top: '445px', right: '54px'},
      { top: '547px', right: '25px'},
      { top: '594px', right: '141px'},
    ],
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    $('.carousel').carousel({
      interval: 400000
    });
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    moveToNextDemoStage: async function() {
      if(this.demoStage === 1){
        this.demoStage = 2;
        await this.forceRender();
        this.setupStageOne();
      } else if(this.demoStage === 3) {
        this.demoStage = 4;
        await this.forceRender();
        this.setupStageTwo();
      } else {
        this.demoStage++;
      }
    },

    setupStageOne: function() {
      this.dragElements.fileOne = $('[purpose="file-one"]')[0];
      this.dragElements.fileTwo = $('[purpose="file-two"]')[0];
      this.dragElements.fileThree = $('[purpose="file-three"]')[0];
      this.dropTargets.hostOne = $('[purpose="host-one"]')[0];
      this.dropTargets.hostTwo = $('[purpose="host-two"]')[0];
      this.dropTargets.hostThree = $('[purpose="host-three"]')[0];
      this.dragElements.fileOne.addEventListener('dragstart', this.dragFile);
      this.dragElements.fileOne.addEventListener('dragend', this.dropFile);
      this.dragElements.fileTwo.addEventListener('dragstart', this.dragFile);
      this.dragElements.fileTwo.addEventListener('dragend', this.dropFile);
      this.dragElements.fileThree.addEventListener('dragstart', this.dragFile);
      this.dragElements.fileThree.addEventListener('dragend', this.dropFile);
      this.dropTargets.hostOne.addEventListener('dragover', this.dragOverTarget);
      this.dropTargets.hostOne.addEventListener('drop', this.dropFileOnHostOne);
      this.dropTargets.hostTwo.addEventListener('dragover', this.dragOverTarget);
      this.dropTargets.hostTwo.addEventListener('drop', this.dropFileOnHostTwo);
      this.dropTargets.hostThree.addEventListener('dragover', this.dragOverTarget);
      this.dropTargets.hostThree.addEventListener('drop', this.dropFileOnHostThree);
      this.startDemoTimer();
    },

    setupStageTwo: function() {
      this.dragElements.fileOne = $('[purpose="file-one"]')[0];
      this.dragElements.fileTwo = $('[purpose="file-two"]')[0];
      this.dragElements.fileThree = $('[purpose="file-three"]')[0];
      this.dropTargets.gitops = $('[purpose="gitops"]')[0];
      this.dragElements.fileOne.addEventListener('dragstart', this.dragFile);
      this.dragElements.fileOne.addEventListener('dragend', this.dropFile);
      this.dragElements.fileTwo.addEventListener('dragstart', this.dragFile);
      this.dragElements.fileTwo.addEventListener('dragend', this.dropFile);
      this.dragElements.fileThree.addEventListener('dragstart', this.dragFile);
      this.dragElements.fileThree.addEventListener('dragend', this.dropFile);
      this.dropTargets.gitops.addEventListener('dragover', this.dragOverTarget);
      this.dropTargets.gitops.addEventListener('drop', this.dropFileOnGitops);
    },

    startDemoTimer: async function() {
      this.timeLeft = this.gameDurationInSeconds;
      let timer = setInterval(() => {
        if (this.timeLeft > 0) {
          this.timeLeft--;
        } else {
          clearInterval(timer);
          if(this.demoStage === 2){
            this.moveToNextDemoStage();
          }
        }
      }, 1000);
    },

    clickStartStageFour: function() {
      this.showStageFourInstructions = false;
      this.startDemoTimer();
    },

    nextGameStage: function() {
      if(this.demoStage <= 4) {
        this.demoStage++;
      }
    },

    clickOpenChatWidget: function() {
      if(window.HubSpotConversations && window.HubSpotConversations.widget){
        window.HubSpotConversations.widget.open();
      }
    },

    dragFile: function(event) {
      event.dataTransfer.setData('text/plain', event.target.id);
      event.target.style.opacity = '0';
    },

    dropFile: function(event) {
      event.target.style.opacity = '1';
    },

    dragOverTarget: function(event) {
      event.preventDefault();
    },

    dropFileOnHostOne: async function(event) {
      event.preventDefault();
      let data = event.dataTransfer.getData('text/plain');
      if(data === 'fileThree'){
        this.counter.hostOne++;
      }
      return;
    },

    dropFileOnHostTwo: async function(event) {
      event.preventDefault();
      let data = event.dataTransfer.getData('text/plain');
      if(data === 'fileOne'){
        this.counter.hostTwo++;
      }
      return;
    },

    dropFileOnHostThree: async function(event) {
      event.preventDefault();
      let data = event.dataTransfer.getData('text/plain');
      if(data === 'fileTwo'){
        this.counter.hostThree++;
      }
      return;
    },

    dropFileOnGitops: async function(event) {
      event.preventDefault();
      let fileId = event.dataTransfer.getData('text/plain');
      let fileToDisappear = document.getElementById(fileId);

      fileToDisappear.classList.add('deploying');
      this.counter.gitOps++;
      // After the animation ends, remove the element from the page.
      fileToDisappear.addEventListener('animationend', ()=>{
        fileToDisappear.parentNode.removeChild(fileToDisappear);
        this.showDeployingMsg = false;
        if(this.counter.gitOps === 3){
          this.showStageFourDeployButton = true;
        }
      });
    },

    clickApproveStageFourChanges: async function() {
      this.showStageFourDeployButton = false;
      this.showDeployingMsg = true;
      await setTimeout( async()=>{
        for (let image of $('[purpose="deployed-linux-file-image"]')) {
          let position = this.stageFourDeployedLinuxFilesFinalPositions[_.indexOf($('[purpose="deployed-linux-file-image"]'), image)];
          $(image).css({
            right: position.right,
            top: position.top,
          });
          let randomNumberOfMilliseconds = Math.floor(Math.random() * (1201)) + 1000;
          await setTimeout(()=>{
            $(image).css({
              animation: 'blinkFade 1s linear',
              opacity: 0,
            });
          }, randomNumberOfMilliseconds);
        }
      }, 1500);
      await setTimeout( async()=>{
        for (let image of $('[purpose="deployed-mac-file-image"]')) {
          let position = this.stageFourDeployedMacFilesFinalPositions[_.indexOf($('[purpose="deployed-mac-file-image"]'), image)];
          $(image).css({
            right: position.right,
            top: position.top,
          });
          let randomNumberOfMilliseconds = Math.floor(Math.random() * (1201)) + 1000;
          await setTimeout(()=>{
            $(image).css({
              animation: 'blinkFade 1s linear',
              opacity: 0,
            });

          }, randomNumberOfMilliseconds);
        }

      }, 1750);
      await setTimeout( async()=>{
        for (let image of $('[purpose="deployed-windows-file-image"]')) {
          let position = this.stageFourDeployedWindowsFilesFinalPositions[_.indexOf($('[purpose="deployed-windows-file-image"]'), image)];
          $(image).css({
            right: position.right,
            top: position.top,
          });
          let randomNumberOfMilliseconds = Math.floor(Math.random() * (1201)) + 1000;
          await setTimeout(()=>{
            $(image).css({
              animation: 'blinkFade 1s linear',
              opacity: 0
            });
          }, randomNumberOfMilliseconds);
        }
      }, 1100);
      await setTimeout( ()=>{
        this.isGameFinished();
      }, 4000);
    },

    isGameFinished: async function() {
      if(this.counter.gitOps === 3){
        this.finalFinishingTime = this.timeLeft;
        this.showSuccessMessage = true;
      }
    },

    doNothing: function() {
      return;
    }

  }
});
