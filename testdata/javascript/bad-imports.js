// Bad: Importing from restricted modules

import { something } from 'lodash';
import _ from 'lodash';
const moment = require('moment');

// These should be caught by no-restricted-imports
import React from 'react';
import { useState } from 'react';

export default function Component() {
  return null;
}
