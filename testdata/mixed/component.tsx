// TypeScript React component for testing TSX

import React, { Component } from 'react';

interface ButtonProps {
  label: string;
  onClick?: () => void;
  disabled?: boolean;
}

interface ButtonState {
  clicked: boolean;
}

class Button extends Component<ButtonProps, ButtonState> {
  constructor(props: ButtonProps) {
    super(props);
    this.state = {
      clicked: false
    };
  }

  handleClick = (): void => {
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
