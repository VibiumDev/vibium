import pytest


@pytest.mark.capability("core")
def test_selected_core():
    pass


@pytest.mark.capability("audio")
def test_no_engine_capability():
    pass


@pytest.mark.capability("core", "audio")
def test_and_requirements():
    pass
