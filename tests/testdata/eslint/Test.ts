// Test.ts - ESLint naming/max-len violation test file
// Using JavaScript-compatible syntax for ESLint without TypeScript parser

// [eslint id-match 위반] snake_case 클래스명 (PascalCase여야 함)
class user_service {
    constructor() {
        // [eslint id-match 위반] snake_case 변수명
        this.api_url = 'https://api.example.com';
    }

    // [eslint id-match 위반] snake_case 함수명
    async fetch_user_data(userId) {
        const response = await fetch(this.api_url + '/users/' + userId);
        const data = await response.json();
        return data;
    }

    // [eslint id-match 위반] snake_case 함수명
    get_user_name(user) {
        return user.name;
    }
}

// [eslint max-len 위반] 100자 초과 라인
const very_long_variable_name_that_exceeds_one_hundred_characters_limit_for_testing_eslint_max_length_rule = 'test';

// [eslint id-match 위반] snake_case 클래스명
class data_processor {
    constructor() {
        // [eslint id-match 위반] snake_case 변수명
        this.processed_count = 0;
    }

    // [eslint id-match 위반] snake_case 함수명
    transform_item(rawItem) {
        return {
            id: rawItem.id,
            name: rawItem.name,
            timestamp: Date.now()
        };
    }
}

// [eslint id-match 위반] snake_case 함수명
async function load_all_data(config) {
    const service = new user_service();
    const result = await service.fetch_user_data(config.userId);
    console.log('Loaded data:', result);
    return result;
}

// [eslint id-match 위반] snake_case 변수명
const api_client = new user_service();
const data_handler = new data_processor();

module.exports = { user_service, data_processor, load_all_data, api_client, data_handler };
