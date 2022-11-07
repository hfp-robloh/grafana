import { rest } from 'msw';
import { SetupServerApi } from 'msw/node';

import {
  ExternalAlertmanagerConfig,
  ExternalAlertmanagersResponse,
} from '../../../../plugins/datasource/alertmanager/types';
import { AlertmanagersChoiceResponse } from '../api/alertmanagerApi';

export function mockAlertmanagerChoiceResponse(server: SetupServerApi, respose: AlertmanagersChoiceResponse) {
  server.use(rest.get('/api/v1/ngalert', (req, res, ctx) => res(ctx.status(200), ctx.json(respose))));
}

export function mockAlertmanagersResponse(server: SetupServerApi, response: ExternalAlertmanagersResponse) {
  server.use(rest.get('/api/v1/ngalert/alertmanagers', (req, res, ctx) => res(ctx.status(200), ctx.json(response))));
}

export function mockAlertmanagerConfigResponse(server: SetupServerApi, response: ExternalAlertmanagerConfig) {
  server.use(rest.get('/api/v1/ngalert/admin_config', (req, res, ctx) => res(ctx.status(200), ctx.json(response))));
}
