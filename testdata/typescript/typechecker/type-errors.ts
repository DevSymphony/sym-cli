// File with TypeScript type errors

interface Person {
  name: string;
  age: number;
}

// Error: Type 'string' is not assignable to type 'number'
const person: Person = {
  name: 'John',
  age: 'thirty' // Should be number
};

// Error: Property 'email' does not exist on type 'Person'
function printEmail(p: Person) {
  console.log(p.email); // email doesn't exist
}

// Error: Cannot find name 'undefinedVariable'
const result = undefinedVariable + 10;

// Error: Argument of type 'number' is not assignable to parameter of type 'string'
function greet(name: string): string {
  return `Hello, ${name}`;
}
greet(123); // Should be string

// Error: Object is possibly 'null'
function getLength(str: string | null) {
  return str.length; // str might be null
}

// Error: 'this' implicitly has type 'any'
const obj = {
  value: 10,
  getValue: function() {
    return function() {
      return this.value; // 'this' has wrong context
    };
  }
};
