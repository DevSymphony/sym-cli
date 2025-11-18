// Core engine module - Developer CANNOT modify this file
class Engine {
  constructor(config) {
    this.config = config;
    this.running = false;
  }

  start() {
    this.running = true;
    console.log("Engine started");
  }

  stop() {
    this.running = false;
    console.log("Engine stopped");
  }
}

module.exports = Engine;
