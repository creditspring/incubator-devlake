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

import { CaretRightOutlined } from '@ant-design/icons';
import { theme, Collapse, Input, InputNumber } from 'antd';

interface Props {
  entities: string[];
  transformation: any;
  setTransformation: React.Dispatch<React.SetStateAction<any>>;
}

export const BasecampTransformation = ({ entities, transformation, setTransformation }: Props) => {
  const { token } = theme.useToken();

  const panelStyle: React.CSSProperties = {
    marginBottom: 24,
    background: token.colorFillAlter,
    borderRadius: token.borderRadiusLG,
    border: 'none',
  };

  return (
    <Collapse
      bordered={false}
      defaultActiveKey={['TICKET']}
      expandIcon={({ isActive }) => <CaretRightOutlined rotate={isActive ? 90 : 0} rev="" />}
      style={{ background: token.colorBgContainer }}
      size="large"
      items={[
        {
          key: 'TICKET',
          label: 'Todos',
          style: panelStyle,
          children: (
            <>
              <h3 style={{ marginBottom: 16 }}>Project Age Limit</h3>
              <p style={{ marginBottom: 16 }}>
                Only sync todos from projects created within the last N months. Set to 0 to sync all projects regardless of age.
              </p>
              <div style={{ marginBottom: 24 }}>
                <InputNumber
                  min={0}
                  max={120}
                  value={transformation.projectAgeLimitMonths ?? 18}
                  onChange={(value) =>
                    setTransformation({
                      ...transformation,
                      projectAgeLimitMonths: value ?? 18,
                    })
                  }
                  addonAfter="months"
                  style={{ width: 150 }}
                />
              </div>

              <h3 style={{ marginBottom: 16 }}>Todo Age Limit</h3>
              <p style={{ marginBottom: 16 }}>
                Only sync todos that were updated within the last N months. Set to 0 to sync all todos regardless of age.
              </p>
              <div style={{ marginBottom: 24 }}>
                <InputNumber
                  min={0}
                  max={120}
                  value={transformation.todoAgeLimitMonths ?? 0}
                  onChange={(value) =>
                    setTransformation({
                      ...transformation,
                      todoAgeLimitMonths: value ?? 0,
                    })
                  }
                  addonAfter="months"
                  style={{ width: 150 }}
                />
              </div>

              <h3 style={{ marginBottom: 16 }}>Permanent Projects</h3>
              <p style={{ marginBottom: 16 }}>
                Enter project IDs that should always sync todos, regardless of the age limit above.
              </p>
              <Input.TextArea
                placeholder="42930251, 12345678"
                value={transformation.permanentProjectIds ?? ''}
                onChange={(e) =>
                  setTransformation({
                    ...transformation,
                    permanentProjectIds: e.target.value,
                  })
                }
                rows={3}
              />
              <p style={{ marginTop: 8, color: token.colorTextSecondary, fontSize: 12 }}>
                Find project IDs in Basecamp URLs: https://3.basecamp.com/&#123;account&#125;/buckets/&#123;PROJECT_ID&#125;/...
              </p>
            </>
          ),
        },
      ].filter((it) => entities.includes(it.key))}
    />
  );
};
