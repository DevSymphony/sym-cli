// This file contains intentional violations for testing

// ❌ Class name should be PascalCase
class myClass {
  constructor() {
    this.value = 42;
  }
}

// ❌ Function name should be camelCase
function MyFunction() {
  return "hello";
}

// ❌ Variable with underscore (if pattern requires camelCase)
const my_variable = 123;

// ❌ Line exceeds 100 characters
const veryLongLine = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore";

// ❌ Function with too many parameters (>5)
function tooManyParams(a, b, c, d, e, f, g) {
  return a + b + c + d + e + f + g;
}

// ✅ Good naming
class User {
  getName() {
    return "John";
  }
}
