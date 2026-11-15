import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


@unittest.skipUnless(shutil.which('pwsh'), 'PowerShell is required')
class DeploymentResponseTests(unittest.TestCase):
    def test_nested_create_and_get_response_and_creation_receipt(self):
        script = Path(__file__).with_name('deploy-harness.ps1').resolve()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            plan = {'model': {'bedrockModelConfig': {'modelId': 'unit-test'}},
                    'executionRoleArn': 'arn:aws:iam::000000000000:role/test',
                    'maxTokens': 2048, 'maxIterations': 6, 'timeoutSeconds': 60,
                    'allowedTools': ['test_read'], 'environment': {'agentCoreRuntimeEnvironment': {
                        'lifecycleConfiguration': {'idleRuntimeSessionTimeout': 60, 'maxLifetime': 300}}}}
            (root/'plan.json').write_text(json.dumps(plan))
            quote = lambda path: "'"+str(path).replace("'", "''")+"'"
            wrapper = """function global:aws {
 $global:LASTEXITCODE=0
 if ($args -contains 'create-harness') { return '{"harness":{"harnessId":"unit-test","status":"CREATING"}}' }
 if ($args -contains 'get-harness') {
  if ($args -notcontains 'unit-test') { throw 'Missing nested harness ID' }
  return '{"harness":{"harnessId":"unit-test","status":"READY"}}'
 }
 throw 'Unexpected AWS operation'
}
"""
            wrapper += '& '+quote(script)+' -InputFile '+quote(root/'plan.json')+' -Region test-region -OutputFile '+quote(root/'state.json')
            (root/'test.ps1').write_text(wrapper)
            result = subprocess.run(['pwsh', '-NoProfile', '-File', str(root/'test.ps1')], capture_output=True, text=True, timeout=30)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads((root/'state.json').read_text(encoding='utf-8-sig'))['harness']['status'], 'READY')
            self.assertEqual(json.loads((root/'state.json.created.json').read_text(encoding='utf-8-sig'))['harness']['harnessId'], 'unit-test')


if __name__ == '__main__':
    unittest.main()
