/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { IPluginConfig } from '@/types';

import Icon from './assets/icon.svg?react';
import { AccountId, ClientId, ClientSecret, RefreshToken } from './connection-fields';

export const BasecampConfig: IPluginConfig = {
  plugin: 'basecamp',
  name: 'Basecamp',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 4,
  isBeta: true,
  connection: {
    docLink: 'https://devlake.apache.org/docs/Configuration/Basecamp',
    initialValues: {
      endpoint: 'https://3.basecampapi.com/',
    },
    fields: [
      'name',
      {
        key: 'endpoint',
        multipleVersions: {
          cloud: 'https://3.basecampapi.com/',
          server: '',
        },
      },
      ({ initialValues, values, errors, setValues, setErrors }: any) => (
        <AccountId
          key="accountId"
          initialValue={initialValues.accountId ?? ''}
          value={values.accountId ?? ''}
          error={errors.accountId ?? ''}
          setValue={(value) => setValues({ accountId: value })}
          setError={(error) => setErrors({ accountId: error })}
        />
      ),
      {
        key: 'token',
        label: 'Access Token',
        subLabel:
          'OAuth2 access token for Basecamp API. Create an integration at https://launchpad.37signals.com/integrations',
      },
      ({ initialValues, values, setValues }: any) => (
        <ClientId
          key="clientId"
          initialValue={initialValues.clientId ?? ''}
          value={values.clientId ?? ''}
          setValue={(value) => setValues({ clientId: value })}
        />
      ),
      ({ type, initialValues, values, setValues }: any) => (
        <ClientSecret
          key="clientSecret"
          type={type}
          initialValue={initialValues.clientSecret ?? ''}
          value={values.clientSecret ?? ''}
          setValue={(value) => setValues({ clientSecret: value })}
        />
      ),
      ({ type, initialValues, values, setValues }: any) => (
        <RefreshToken
          key="refreshToken"
          type={type}
          initialValue={initialValues.refreshToken ?? ''}
          value={values.refreshToken ?? ''}
          setValue={(value) => setValues({ refreshToken: value })}
        />
      ),
      'proxy',
      {
        key: 'rateLimitPerHour',
        subLabel:
          'By default, DevLake uses 1,000 requests/hour for Basecamp. Basecamp has a rate limit of ~50 requests per 10 seconds.',
        defaultValue: 1000,
      },
    ],
  },
  dataScope: {
    title: 'Accounts',
  },
  scopeConfig: {
    entities: ['TICKET', 'CROSS'],
    transformation: {
      projectAgeLimitMonths: 18,
      todoAgeLimitMonths: 0,
      permanentProjectIds: '',
      documentVaultUrls: '',
    },
  },
};
