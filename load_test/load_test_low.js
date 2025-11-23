import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
	stages: [
		{ duration: '1m', target: 5 },
	],
	thresholds: {
		http_req_duration: ['p(95)<300'],
		errors: ['rate<0.001'],
		checks: ['rate>0.99'],
	},
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
	const teamName = `team-${__VU}-${__ITER}`;
	const userID = `u${__VU}${__ITER}`;
	const prID = `pr-${__VU}${__ITER}`;

	const teamPayload = JSON.stringify({
		team_name: teamName,
		members: [
			{ user_id: userID, username: `User-${__VU}-${__ITER}`, is_active: true },
		],
	});

	const createTeamRes = http.post(`${BASE_URL}/team/add`, teamPayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const createTeamOk = check(createTeamRes, {
		'POST /team/add status is 201 or 400': (r) => r.status === 201 || r.status === 400,
		'POST /team/add response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!createTeamOk);
	sleep(0.2);

	const getTeamRes = http.get(`${BASE_URL}/team/get?team_name=${teamName}`);
	const getTeamOk = check(getTeamRes, {
		'GET /team/get status is 200 or 404': (r) => r.status === 200 || r.status === 404,
		'GET /team/get response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!getTeamOk);
	sleep(0.2);

	const setIsActivePayload = JSON.stringify({
		user_id: userID,
		is_active: false,
	});
	const setIsActiveRes = http.post(`${BASE_URL}/users/setIsActive`, setIsActivePayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const setIsActiveOk = check(setIsActiveRes, {
		'POST /users/setIsActive status is 200 or 404': (r) => r.status === 200 || r.status === 404,
		'POST /users/setIsActive response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!setIsActiveOk);
	sleep(0.2);

	const getReviewRes = http.get(`${BASE_URL}/users/getReview?user_id=${userID}`);
	const getReviewOk = check(getReviewRes, {
		'GET /users/getReview status is 200 or 404': (r) => r.status === 200 || r.status === 404,
		'GET /users/getReview response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!getReviewOk);
	sleep(0.2);

	const createPRPayload = JSON.stringify({
		pull_request_id: prID,
		pull_request_name: `PR-${__VU}-${__ITER}`,
		author_id: userID,
	});
	const createPRRes = http.post(`${BASE_URL}/pullRequest/create`, createPRPayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const createPROk = check(createPRRes, {
		'POST /pullRequest/create status is 201 or 404 or 409': (r) => r.status === 201 || r.status === 404 || r.status === 409,
		'POST /pullRequest/create response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!createPROk);
	sleep(0.2);

	const mergePRPayload = JSON.stringify({
		pull_request_id: prID,
	});
	const mergePRRes = http.post(`${BASE_URL}/pullRequest/merge`, mergePRPayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const mergePROk = check(mergePRRes, {
		'POST /pullRequest/merge status is 200 or 404': (r) => r.status === 200 || r.status === 404,
		'POST /pullRequest/merge response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!mergePROk);
	sleep(0.2);

	const reassignPayload = JSON.stringify({
		pull_request_id: prID,
		old_user_id: userID,
	});
	const reassignRes = http.post(`${BASE_URL}/pullRequest/reassign`, reassignPayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const reassignOk = check(reassignRes, {
		'POST /pullRequest/reassign status is 200 or 404 or 409': (r) => r.status === 200 || r.status === 404 || r.status === 409,
		'POST /pullRequest/reassign response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!reassignOk);
	sleep(0.2);

	const statsRes = http.get(`${BASE_URL}/stats`);
	const statsOk = check(statsRes, {
		'GET /stats status is 200': (r) => r.status === 200,
		'GET /stats response time < 300ms': (r) => r.timings.duration < 300,
	});
	errorRate.add(!statsOk);
	sleep(0.2);
}

