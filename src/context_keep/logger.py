# src/context_keep/logging.py
import logging


def setup_logger():
    """Sets up a logger for Context-Keep."""
    logger = logging.getLogger("context_keep")
    logger.setLevel(logging.INFO)

    handler = logging.StreamHandler()
    formatter = logging.Formatter("%(asctime)s - %(levelname)s - %(message)s")
    handler.setFormatter(formatter)

    logger.addHandler(handler)
    return logger


logger = setup_logger()
