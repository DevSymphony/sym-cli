// Button component - Developer can modify this file
class Button {
  constructor(label) {
    this.label = label;
  }

  render() {
    return `<button>${this.label}</button>`;
  }
}

module.exports = Button;
