import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
  stages: [
    { duration: '30s', target: 20 }, // Ramp-up to 20 users
    { duration: '1m', target: 20 },  // Stay at 20 users for 1 minute
    { duration: '30s', target: 0 },  // Ramp-down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2s
    http_req_failed: ['rate<0.01'],    // Less than 1% of requests should fail
  },
};

const BASE_URL = 'http://localhost:8090'; // Adjust this to your service URL
let token = ''; // You'll need to set this with a valid Super Admin token
let createdRoleId = 0;

export function setup() {
  // Here you would typically get your authentication token
  // For this example, we'll assume you have a Super Admin token
  return { token: 'YOUR_SUPER_ADMIN_JWT_TOKEN' };
}

export default function (data) {
  token = data.token;
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };

  // 1. Create Role Test
  const createPayload = JSON.stringify({
    name: `Test Role ${randomString(8)}`,
  });

  const createRes = http.post(`${BASE_URL}/api/roles`, createPayload, {
    headers: headers,
  });

  check(createRes, {
    'Create role status is 201': (r) => r.status === 201,
    'Create response has success message': (r) => r.json('message') === 'Success',
  });

  if (createRes.status === 201) {
    // Store the created role ID if available in response
    // Note: Adjust this based on your actual response structure
    createdRoleId = createRes.json('data.id');
  }

  sleep(1);

  // 2. Get All Roles Test
  const getAllRes = http.get(`${BASE_URL}/api/roles`, {
    headers: headers,
  });

  check(getAllRes, {
    'Get all roles status is 200': (r) => r.status === 200,
    'Get all roles returns array': (r) => Array.isArray(r.json('data')),
  });

  sleep(1);

  // 3. Get Role by ID Test
  if (createdRoleId) {
    const getByIdRes = http.get(`${BASE_URL}/api/roles/${createdRoleId}`, {
      headers: headers,
    });

    check(getByIdRes, {
      'Get role by ID status is 200': (r) => r.status === 200,
      'Get role by ID returns correct data': (r) => r.json('data') !== null,
    });
  }

  sleep(1);

  // 4. Update Role Test
  if (createdRoleId) {
    const updatePayload = JSON.stringify({
      name: `Updated Role ${randomString(8)}`,
    });

    const updateRes = http.put(`${BASE_URL}/api/roles/${createdRoleId}`, updatePayload, {
      headers: headers,
    });

    check(updateRes, {
      'Update role status is 200': (r) => r.status === 200,
      'Update response has success message': (r) => r.json('message') === 'Success',
    });
  }

  sleep(1);

  // 5. Delete Role Test
  if (createdRoleId) {
    const deleteRes = http.del(`${BASE_URL}/api/roles/${createdRoleId}`, null, {
      headers: headers,
    });

    check(deleteRes, {
      'Delete role status is 200': (r) => r.status === 200,
      'Delete response has success message': (r) => r.json('message') === 'Role deleted successfully',
    });
  }

  sleep(1);
}
