#!/usr/bin/env python
# -*- coding: utf-8 -*-

try:
    import setuptools
except ImportError:
    import distutils.core as setuptools


__copyright__ = 'Copyright 2015'
__version__ = '0.1'

__title__ = 'docker-registry-speedy-driver'
__build__ = 0x000000

__description__ = 'Docker registry speedy driver'

setuptools.setup(
    name=__title__,
    version=__version__,
    description=__description__,
    platforms=['Independent'],
    #namespace_packages=['docker_registry', 'docker_registry.drivers'],
    packages=['docker_registry', 'docker_registry.drivers'],
    #packages=setuptools.find_packages(),
    zip_safe=True,
)
