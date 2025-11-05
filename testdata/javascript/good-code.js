// This file follows all conventions

class UserService {
  constructor() {
    this.users = [];
  }

  addUser(user) {
    this.users.push(user);
  }

  getUsers() {
    return this.users;
  }
}

function calculateTotal(items) {
  return items.reduce((sum, item) => sum + item.price, 0);
}

const apiEndpoint = "https://api.example.com";

module.exports = { UserService, calculateTotal };
