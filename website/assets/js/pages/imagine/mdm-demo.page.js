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
    timeLeft: 25,
    showDeployingMsg: false,
    formData: {},
    formErrors: {},
    formRules: {},
    syncing: false,
    cloudError: false,
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
        this.startDemoTimer();
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
      this.startDemoTimer();
    },

    startDemoTimer() {
      this.timeLeft = 25;
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

    nextGameStage: function(){
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
      var data = event.dataTransfer.getData('text/plain');
      if(data === 'fileThree'){
        this.counter.hostOne++;
      }
      return;
    },

    dropFileOnHostTwo: async function(event) {
      event.preventDefault();
      var data = event.dataTransfer.getData('text/plain');
      console.log(data);
      if(data === 'fileOne'){
        this.counter.hostTwo++;
      }
      return;
    },

    dropFileOnHostThree: async function(event) {
      event.preventDefault();
      var data = event.dataTransfer.getData('text/plain');
      console.log(data);
      if(data === 'fileTwo'){
        this.counter.hostThree++;
      }
      return;
    },

    dropFileOnGitops: async function(event) {
      event.preventDefault();
      var data = event.dataTransfer.getData('text/plain');
      let fileToDisappear = document.getElementById(data);

      fileToDisappear.classList.add('deploying');
      this.showDeployingMsg = true;
      let arrow;
      if(data === 'windowsFile'){
        arrow = $('[purpose="arrow-two"]')[0];
        arrow.style.animation = 'blinkFade 1s linear';
        this.counter.gitOps++;
      } else if(data === 'macFile') {
        arrow = $('[purpose="arrow-three"]')[0];
        arrow.style.animation = 'blinkFade 1s linear';
        this.counter.gitOps++;
      } else if(data === 'linuxFile') {
        arrow = $('[purpose="arrow-one"]')[0];
        arrow.style.animation = 'blinkFade 1s linear';
        this.counter.gitOps++;
      }
      // After the animation ends, remove the element from the page.
      fileToDisappear.addEventListener('animationend', () => {
        fileToDisappear.parentNode.removeChild(fileToDisappear);
        this.showDeployingMsg = false;
      });
      await this.isGameFinished();
    },

    isGameFinished: async function() {
      if(this.counter.gitOps === 3){
        await setTimeout(()=>{
          this.moveToNextDemoStage();
        }, 2000);
      }
    },

    doNothing: function() {
      return;
    }

  }
});
