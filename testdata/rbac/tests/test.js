// Test file - Developer can modify this file
const Button = require('../src/components/Button');

describe('Button', () => {
  it('should render button with label', () => {
    const button = new Button('Click me');
    const html = button.render();
    expect(html).toBe('<button>Click me</button>');
  });
});
