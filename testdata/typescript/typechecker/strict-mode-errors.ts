// File with strict mode violations

// Error: Variable 'x' implicitly has an 'any' type
let x;
x = 10;
x = 'string';

// Error: Parameter 'input' implicitly has an 'any' type
function processInput(input) {
  return input.toUpperCase();
}

// Error: Function lacks return type annotation
function calculate(a: number, b: number) {
  return a + b;
}

// Error: Object is possibly 'undefined'
function getUserName(user: { name?: string }) {
  return user.name.toUpperCase(); // name might be undefined
}

// Error: Not all code paths return a value
function getValue(condition: boolean): string {
  if (condition) {
    return 'yes';
  }
  // Missing return for false case
}
