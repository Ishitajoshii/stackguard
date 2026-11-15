import tempfile
import unittest
from pathlib import Path
from lab_env import tokens


class LabEnvTests(unittest.TestCase):
    def parse(self, text):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'lab.env'
            path.write_text(text)
            return tokens(path)

    def test_only_requested_keys_and_no_shell_evaluation(self):
        value = self.parse('UNRELATED_SECRET=excluded\nSTACKGUARD_GITHUB_TOKEN="$(not-a-command)"\nSTACKGUARD_NOTION_TOKEN=local-value\n')
        self.assertEqual(set(value), {'STACKGUARD_GITHUB_TOKEN', 'STACKGUARD_NOTION_TOKEN'})
        self.assertEqual(value['STACKGUARD_GITHUB_TOKEN'], '$(not-a-command)')

    def test_duplicate_and_missing_fail_without_disclosing_values(self):
        for text in ('STACKGUARD_GITHUB_TOKEN=private-value\n',
                     'STACKGUARD_GITHUB_TOKEN=private-value\nSTACKGUARD_GITHUB_TOKEN=second-private\nSTACKGUARD_NOTION_TOKEN=value\n'):
            with self.assertRaises(ValueError) as context:
                self.parse(text)
            self.assertNotIn('private', str(context.exception))


if __name__ == '__main__':
    unittest.main()
