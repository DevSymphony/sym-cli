// Test.ts - Prettier formatting violation test file

// [prettier 위반] 4칸 들여쓰기 (2칸이어야 함)
class UserService {
    // [prettier 위반] 큰따옴표 사용 (작은따옴표여야 함)
    private apiUrl: string = "https://api.example.com";

    async fetchUserData(userId: string) {
        // [prettier 위반] 큰따옴표 사용
        const response = await fetch(this.apiUrl + "/users/" + userId);
        const data = await response.json();
        return data;
    }

    getUserName(user: { name: string }): string {
        return user.name;
    }
}

// [prettier 위반] 100자 초과 라인
const veryLongVariableNameThatExceedsOneHundredCharactersLimitForTestingPrettierPrintWidthRule: string = "this is a very long value that makes this line exceed the maximum allowed characters for prettier";

class DataProcessor {
    private processedCount: number = 0;

    transformItem(rawItem: { id: number; name: string }) {
        return {
            id: rawItem.id,
            // [prettier 위반] 큰따옴표 사용
            name: rawItem.name + " processed",
            timestamp: Date.now()
        };
    }
}

async function loadAllData(config: { userId: string }) {
    const service = new UserService();
    const result = await service.fetchUserData(config.userId);
    // [prettier 위반] 큰따옴표 사용
    console.log("Loaded data:", result);
    return result;
}

const apiClient = new UserService();
const dataHandler = new DataProcessor();

export { UserService, DataProcessor, loadAllData, apiClient, dataHandler };
