// Test.ts - TypeScript strict null checks violation test file

class UserService {
  private apiUrl: string = 'https://api.example.com';

  async fetchUserData(userId: string) {
    const response = await fetch(this.apiUrl + '/users/' + userId);
    const data = await response.json();
    return data;
  }

  // [tsc strictNullChecks 위반] undefined 가능 값을 string으로 반환
  getUserName(user: { name?: string }): string {
    return user.name;  // user.name이 undefined일 수 있음
  }

  // [tsc strictNullChecks 위반] undefined 가능 값을 number로 반환
  getUserAge(user: { age?: number }): number {
    return user.age;  // user.age가 undefined일 수 있음
  }
}

class DataProcessor {
  private processedCount: number = 0;

  // [tsc strictNullChecks 위반] undefined 가능 값을 number로 반환
  getItemId(item: { id?: number }): number {
    return item.id;  // item.id가 undefined일 수 있음
  }

  // [tsc strictNullChecks 위반] null 가능 값을 string으로 반환
  getItemName(item: { name: string | null }): string {
    return item.name;  // item.name이 null일 수 있음
  }
}

const userService = new UserService();
const dataProcessor = new DataProcessor();

export { UserService, DataProcessor, userService, dataProcessor };
