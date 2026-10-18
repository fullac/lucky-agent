import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("agent_graph", Path(__file__).with_name("verify-agent-graph.py"))
graph = importlib.util.module_from_spec(spec)
spec.loader.exec_module(graph)


class AgentGraphContract(unittest.TestCase):
    def test_detects_production_and_manifest_differences(self):
        with tempfile.TemporaryDirectory() as directory:
            source, selected = Path(directory) / "source", Path(directory) / "selected"
            for root in (source, selected):
                root.mkdir()
                (root / "go.mod").write_text("module fixture\n")
                (root / "core.go").write_text("package fixture\n")
            self.assertEqual(graph.compare(source, selected), [])
            (source / "core.go").write_text("package changed\n")
            (source / "new.go").write_text("package fixture\n")
            (selected / "go.mod").write_text("module different\n")
            self.assertEqual(graph.compare(source, selected), ["core.go", "go.mod", "new.go"])

    def test_ignores_tests_and_docs_but_keeps_runtime_assets(self):
        with tempfile.TemporaryDirectory() as directory:
            source, selected = Path(directory) / "source", Path(directory) / "selected"
            for root in (source, selected):
                root.mkdir()
                (root / "go.mod").write_text("module fixture\n")
                (root / "core.go").write_text("package fixture\n")
            (source / "contract_test.go").write_text("test")
            (source / "README.md").write_text("documentation")
            (source / "docs").mkdir()
            (source / "docs" / "guide.md").write_text("guide")
            self.assertEqual(graph.compare(source, selected), [])
            (source / "prompt.md").write_text("runtime prompt")
            self.assertEqual(graph.compare(source, selected), ["prompt.md"])


if __name__ == "__main__":
    unittest.main()
