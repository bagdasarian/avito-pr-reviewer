import http from 'k6/http';
import { check } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
	stages: [
		{ duration: '15s', target: 100 },
		{ duration: '15s', target: 200 },
		{ duration: '15s', target: 300 },
		{ duration: '15s', target: 500 },
	],
	thresholds: {
		http_req_duration: ['p(95)<2000'],
		errors: ['rate<0.1'],
		checks: ['rate>0.90'],
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
	});
	errorRate.add(!createTeamOk);

	const getTeamRes = http.get(`${BASE_URL}/team/get?team_name=${teamName}`);
	const getTeamOk = check(getTeamRes, {
		'GET /team/get status is 200 or 404': (r) => r.status === 200 || r.status === 404,
	});
	errorRate.add(!getTeamOk);

	const setIsActivePayload = JSON.stringify({
		user_id: userID,
		is_active: false,
	});
	const setIsActiveRes = http.post(`${BASE_URL}/users/setIsActive`, setIsActivePayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const setIsActiveOk = check(setIsActiveRes, {
		'POST /users/setIsActive status is 200 or 404': (r) => r.status === 200 || r.status === 404,
	});
	errorRate.add(!setIsActiveOk);

	const getReviewRes = http.get(`${BASE_URL}/users/getReview?user_id=${userID}`);
	const getReviewOk = check(getReviewRes, {
		'GET /users/getReview status is 200 or 404': (r) => r.status === 200 || r.status === 404,
	});
	errorRate.add(!getReviewOk);

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
	});
	errorRate.add(!createPROk);

	const mergePRPayload = JSON.stringify({
		pull_request_id: prID,
	});
	const mergePRRes = http.post(`${BASE_URL}/pullRequest/merge`, mergePRPayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const mergePROk = check(mergePRRes, {
		'POST /pullRequest/merge status is 200 or 404': (r) => r.status === 200 || r.status === 404,
	});
	errorRate.add(!mergePROk);

	const reassignPayload = JSON.stringify({
		pull_request_id: prID,
		old_user_id: userID,
	});
	const reassignRes = http.post(`${BASE_URL}/pullRequest/reassign`, reassignPayload, {
		headers: { 'Content-Type': 'application/json' },
	});
	const reassignOk = check(reassignRes, {
		'POST /pullRequest/reassign status is 200 or 404 or 409': (r) => r.status === 200 || r.status === 404 || r.status === 409,
	});
	errorRate.add(!reassignOk);

	const statsRes = http.get(`${BASE_URL}/stats`);
	const statsOk = check(statsRes, {
		'GET /stats status is 200': (r) => r.status === 200,
	});
	errorRate.add(!statsOk);
}

