// React component for testing JSX

import React from 'react';

class Button extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      clicked: false
    };
  }

  handleClick = () => {
    this.setState({ clicked: true });
    if (this.props.onClick) {
      this.props.onClick();
    }
  }

  render() {
    const { label, disabled } = this.props;
    const { clicked } = this.state;

    return (
      <button
        onClick={this.handleClick}
        disabled={disabled}
        className={clicked ? 'clicked' : ''}
      >
        {label}
      </button>
    );
  }
}

export default Button;
