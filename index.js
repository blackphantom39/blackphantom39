// index.js to build README.md from mustache template
import Mustache from 'mustache';
import * as fs from 'fs';
import { badgeConfigs } from './resources/badges.js';

//
// ─── Constants ───────────────────────────────────────────────────────────────────
//
const MUSTACHE_MAIN_DIR = './main.mustache'

//
// ─── Data ────────────────────────────────────────────────────────────────────────
//

/**
 * Data provided to the mustache
 * template.
 */
let DATA = {
  name: 'Alex',
  refresh_date: new Date().toLocaleDateString('en-DE', {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric',
    timeZoneName: 'short',
    timeZone: 'Europe/Berlin'
  }),
};

//
// ─── Functions ───────────────────────────────────────────────────────────────────
//

/**
 * Open the Mustache template and render it using
 * the DATA object. Finally write the rendered
 * content to the README file
 */
async function generateReadMe() {
  await fs.readFile(MUSTACHE_MAIN_DIR, (err, data) => {
    if (err) throw err;
    const output = Mustache.render(data.toString(), DATA);
    fs.writeFileSync('README.md', output);
  })
}

/**
 * Calculates the age given the birth date.
 * The age is appended to the DATA object
 *
 * @param {string} dateString The birth date
 */
function getAge(dateString) {
  const today = new Date();
  const birth = new Date(dateString);
  let age = today.getFullYear() - birth.getFullYear();
  const monthDiff = today.getMonth() - birth.getMonth();

  if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birth.getDate())) {
    age--;
  }

  DATA.age = age;
}

/**
 * Generates a dictionary of bages based on 'badges.js' file.
 * The provided badge config data will be converted to the
 * shield.io links and grouped based on the provided value
 * for the property 'group'.
 *
 * @returns Dictionary of badge links by group
 */
function createBadgeDictionary() {
  const grouped = groupBy(badgeConfigs, 'group');
  for (const [key, value] of Object.entries(grouped)) {
    let rendered = value.map((badgeConfig) => ({ badge : generateBadge(badgeConfig)}));
    grouped[key] = rendered;
  }
  return grouped;
}


// --- Utility -------------------------------------------------

/**
 * Gets the list of additional arguments passed
 * to the process. 'node' and the excecuted file
 * are trimmed away.
 *
 * @returns string[] of additional arguments
 */
const getArguments = () => process.argv.slice(2);

/**
 * Creates an dictionary from an array by grouping the provided
 * array by a specified property.
 *
 * @param {[*]} arr The array to group
 * @param {string} property The property to group by
 * @returns An dictionary with the property values as keys and the respective values as lists
 */
const groupBy = (arr, property) => {
  return arr.reduce(function(memo, x) {
    if (!memo[x[property]]) { memo[x[property]] = []; }
    memo[x[property]].push(x);
    return memo;
  }, {});
}

/**
 * Enhances 'encodeURIComponent()' to also encode
 * the characters ! ' ( ) and *.
 *
 * @param {string} str The string to encode
 * @returns The URL encoded string
 */
function fixedEncodeURIComponent(str) {
  return encodeURIComponent(str).replace(/[!'()*]/g, function(c) {
    return '%' + c.charCodeAt(0).toString(16);
  });
}

/**
 * Creates a link for a shields io badge based on the provided values
 *
 * @param {*} badgeConfig The badge config object.
 * @returns The shields.io linkt to display a badge
 */
function generateBadge(badgeConfig) {
  for (const [key, value] of Object.entries(badgeConfig)) {
    badgeConfig[key] = fixedEncodeURIComponent(value);
  }
  const url = `[![${badgeConfig.name} Badge](https://img.shields.io/badge/` +
              `-${badgeConfig.badgeText}` +
              `-${badgeConfig.labelBgColor}` +
              `.svg?style=for-the-badge` +
              `&logo=${badgeConfig.logo}` +
              `&logoColor=${badgeConfig.logoColor}` +
              `&link=${badgeConfig.link})]` +
              `(${badgeConfig.link || 'https://github.com/blackphantom39'})`;

  return url;
}

//
// ─── Execute ─────────────────────────────────────────────────────────────────────
//

/**
 * All actions to run in order to generate
 * the readme.
 */
async function actions() {
  // ToDo: Add more actions like fetching data, etc.

  // Get arguments
  const args = getArguments();

  // Perform actions to gather data for the readme
  getAge(args[0]);

  // Generate Badges
  DATA.badges = createBadgeDictionary()

  // Generate the Readme
  await generateReadMe();
}

// Run actions
actions()