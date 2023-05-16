parasails.registerPage('mdm-demo', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    demoStage: 2,
    startingPositons: {
      fileOne: { top: 50, left: 50},
      fileTwo: { top: 200, left: 50},
      fileThree: { top: 350, left: 50},
    },
    counter: {
      hostOne: 0,
      hostTwo: 0,
      hostThree: 0,
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
    fileDroppedOnCorrectHost: {
      fileOne: undefined,
      fileTwo: undefined,
      fileThree: undefined,
    },
    dragContainer: undefined,
    gameStartsAt: undefined,
    gameEndsAt: undefined,
    timeLeft: 25,
    showDeployingMsg: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    this.dragContainer = document.getElementById("dragContainer");
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickStartGame: async function() {
      this.demoStage = 1;
      await this.forceRender();
      this.setupStageOne();
    },
    clickStartStageTwo: async function() {
      this.demoStage = 3;
      await this.forceRender();
      this.setupStageTwo();
      this.startTimer();
    },

    setupStageOne: function() {
      this.dragElements.fileOne = document.getElementById("fileOne");
      this.dragElements.fileTwo = document.getElementById("fileTwo");
      this.dragElements.fileThree = document.getElementById("fileThree");
      this.dropTargets.hostOne = document.getElementById("hostOne");
      this.dropTargets.hostTwo = document.getElementById("hostTwo");
      this.dropTargets.hostThree = document.getElementById("hostThree");

      this.dragElements.fileOne.addEventListener("dragstart", this.dragStart);
      this.dragElements.fileOne.addEventListener("dragend", this.dragEnd);
      this.dragElements.fileTwo.addEventListener("dragstart", this.dragStart);
      this.dragElements.fileTwo.addEventListener("dragend", this.dragEnd);
      this.dragElements.fileThree.addEventListener("dragstart", this.dragStart);
      this.dragElements.fileThree.addEventListener("dragend", this.dragEnd);
      this.dropTargets.hostOne.addEventListener("dragover", this.dragOver);
      this.dropTargets.hostOne.addEventListener("drop", this.dropFileOnHostOne);
      this.dropTargets.hostTwo.addEventListener("dragover", this.dragOver);
      this.dropTargets.hostTwo.addEventListener("drop", this.dropFileOnHostTwo);
      this.dropTargets.hostThree.addEventListener("dragover", this.dragOver);
      this.dropTargets.hostThree.addEventListener("drop", this.dropFileOnHostThree);
      this.startTimer();
    },
    setupStageTwo: function() {
      this.dragElements.fileOne = document.getElementById("fileOne");
      this.dragElements.fileTwo = document.getElementById("fileTwo");
      this.dragElements.fileThree = document.getElementById("fileThree");
      this.dropTargets.gitops = document.getElementById("gitops");


      this.dragElements.fileOne.addEventListener("dragstart", this.dragStart);
      this.dragElements.fileOne.addEventListener("dragend", this.dragEnd);
      this.dragElements.fileTwo.addEventListener("dragstart", this.dragStart);
      this.dragElements.fileTwo.addEventListener("dragend", this.dragEnd);
      this.dragElements.fileThree.addEventListener("dragstart", this.dragStart);
      this.dragElements.fileThree.addEventListener("dragend", this.dragEnd);

      this.dropTargets.gitops.addEventListener("dragover", this.dragOver);
      this.dropTargets.gitops.addEventListener("drop", this.dropFileOnGitops);
      this.startTimer();
    },
    startTimer() {
      let duration = 25; // Set the duration of the timer in seconds
      this.timeLeft = duration;

      let timer = setInterval(() => {
        if (this.timeLeft > 0) {
          this.timeLeft--;
        } else {
          clearInterval(timer);
          this.nextGameStage();
        }
      }, 1000);
    },
    nextGameStage: function(){
      if(this.demoStage <= 2) {
        this.demoStage++;
      }
    },
    clickOpenChatWidget: function() {
      if(window.HubSpotConversations && window.HubSpotConversations.widget){
        window.HubSpotConversations.widget.open();
      }
    },

    dragStart: function(event) {
      event.dataTransfer.setData("text/plain", event.target.id);
      event.target.style.opacity = "0";
    },

    dragEnd: function(event) {
      var data = event.dataTransfer.getData("text/plain");
      event.target.style.opacity = "1";
    },
    dragEndStageTwo: function(event) {
      var data = event.dataTransfer.getData("text/plain");
      event.target.style.opacity = "1";
    },

    dragOver: function(event) {
      event.preventDefault();
    },

    dropFileOnHostOne: async function(event) {
      event.preventDefault();
      var data = event.dataTransfer.getData("text/plain");
      let fileToDisappear = document.getElementById(data);
      // fileToDisappear.style.animation = "blinkFade 3s linear";
      if(data === 'fileOne'){
        fileToDisappear.classList.add('deploying')
        this.counter.hostOne++;
      }
      return;
    },

    dropFileOnHostTwo: async function(event) {
      event.preventDefault();
      var data = event.dataTransfer.getData("text/plain");
      let fileToDisappear = document.getElementById(data);
      // fileToDisappear.style.animation = "blinkFade 3s linear";
      if(data === 'fileTwo'){
        fileToDisappear.classList.add('deploying')
        this.counter.hostTwo++;
      }
      return;
    },

    dropFileOnHostThree: async function(event) {
      event.preventDefault();
      var data = event.dataTransfer.getData("text/plain");
      let fileToDisappear = document.getElementById(data);
      // fileToDisappear.style.animation = "blinkFade 3s linear";
      if(data === 'fileThree'){
        fileToDisappear.classList.add('deploying')
        this.counter.hostThree++;
      }
      return;
    },
    dropFileOnGitops: async function(event) {
      event.preventDefault();
      var data = event.dataTransfer.getData("text/plain");
      let fileToDisappear = document.getElementById(data);

      fileToDisappear.classList.add('deploying');
      this.showDeployingMsg = true;
      let arrow;
      if(data === 'fileOne'){
        arrow = document.getElementById('arrowTwo');
        arrow.style.animation = "blinkFade 1s linear";
      } else if(data === 'fileTwo') {
        arrow = document.getElementById('arrowThree');
        arrow.style.animation = "blinkFade 1s linear";
      } else if(data === 'fileThree') {
        arrow = document.getElementById('arrowOne');
        arrow.style.animation = "blinkFade 1s linear";
      }
      // After the animation ends, remove the element from the DOM
      fileToDisappear.addEventListener('animationend', () => {
        fileToDisappear.parentNode.removeChild(fileToDisappear);
        this.showDeployingMsg = false;
      });
    },
  }
});
